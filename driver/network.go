package driver

import (
	"bytes"
	"crypto/rand"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/davecgh/go-spew/spew"
	"github.com/digitalocean/go-libvirt"
	"libvirt.org/go/libvirtxml"
)

func (d *LibVirtDriverPlugin) setNetworkInterfaces(client *libvirt.Libvirt, cfg *TaskConfig, domainDef *libvirtxml.Domain) error {

	for _, networkInterface := range cfg.NetworkInterface {

		macAddress, err := randomMACAddress()
		if err != nil {
			return err
		}

		netIface := libvirtxml.DomainInterface{
			Model: &libvirtxml.DomainInterfaceModel{
				Type: "virtio",
			},
			MAC: &libvirtxml.DomainInterfaceMAC{
				Address: macAddress,
			},
		}

		// connect to the interface to the network... first, look for the network
		libvirtNetwork, err := client.NetworkLookupByName(networkInterface.NetworkName)
		if err != nil {
			return fmt.Errorf("failed to lookup network: %v", err)
		}

		if libvirtNetwork.Name != "" {
			networkDef, err := getXMLNetworkDefFromLibvirt(client, libvirtNetwork)
			if err != nil {
				return err
			}

			if hasDHCP(networkDef) {
				if networkInterface.Address != "" {
					ip := net.ParseIP(networkInterface.Address)
					if ip == nil {
						return fmt.Errorf("could not parse addresses '%s'", networkInterface.Address)
					}
					if err := updateOrAddHost(client, libvirtNetwork, ip.String(), macAddress, "something"); err != nil {
						return err
					}
				}
			}
		}
		netIface.Source = &libvirtxml.DomainInterfaceSource{
			Network: &libvirtxml.DomainInterfaceSourceNetwork{
				Network: libvirtNetwork.Name,
			},
		}
		domainDef.Devices.Interfaces = append(domainDef.Devices.Interfaces, netIface)
	}

	return nil
}

func randomMACAddress() (string, error) {
	buf := make([]byte, 3)
	//nolint:gosec // math.rand is enough for this
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	// set local bit and unicast
	buf[0] = (buf[0] | 2) & 0xfe
	// Set the local bit
	buf[0] |= 2

	// avoid libvirt-reserved addresses
	if buf[0] == 0xfe {
		buf[0] = 0xee
	}

	return fmt.Sprintf("52:54:00:%02x:%02x:%02x",
		buf[0], buf[1], buf[2]), nil
}

// HasDHCP checks if the network has a DHCP server managed by libvirt.
func hasDHCP(net libvirtxml.Network) bool {
	if net.Forward != nil {
		if net.Forward.Mode == "nat" || net.Forward.Mode == "route" || net.Forward.Mode == "open" || net.Forward.Mode == "" {
			return true
		}
	} else {
		// isolated network
		return true
	}
	return false
}

func getXMLNetworkDefFromLibvirt(virConn *libvirt.Libvirt, network libvirt.Network) (libvirtxml.Network, error) {
	networkXMLDesc, err := virConn.NetworkGetXMLDesc(network, 0)
	if err != nil {
		return libvirtxml.Network{}, fmt.Errorf("error retrieving libvirt network XML description: %w", err)
	}
	networkDef := libvirtxml.Network{}
	err = xml.Unmarshal([]byte(networkXMLDesc), &networkDef)
	if err != nil {
		return libvirtxml.Network{}, fmt.Errorf("error reading libvirt network XML description: %w", err)
	}
	return networkDef, nil
}

func updateOrAddHost(virConn *libvirt.Libvirt, n libvirt.Network, ip, mac, name string) error {
	xmlNet, _ := getXMLNetworkDefFromLibvirt(virConn, n)
	// We don't check the error above
	// if we can't parse the network to xml for some reason
	// we will return the default '-1' value.
	xmlIdx, err := getNetworkIdx(&xmlNet, ip)
	if err != nil {
		log.Printf("Error during detecting network index: %s\nUsing default value: %d", err, xmlIdx)
	}

	err = updateHost(virConn, n, ip, mac, name, xmlIdx)
	// FIXME: libvirt.Error.DomainID is not available from library. Is it still required here?
	//  && virErr.Error.DomainID == uint32(.....FromNetwork) {
	if isError(err, libvirt.ErrOperationInvalid) {
		log.Printf("[DEBUG]: karl: updateOrAddHost before addHost()\n")
		return addHost(virConn, n, ip, mac, name, xmlIdx)
	}
	return err
}

func getNetworkIdx(n *libvirtxml.Network, ip string) (int, error) {
	xmlIdx := -1

	if n == nil {
		return xmlIdx, fmt.Errorf("failed to convert to libvirt XML")
	}

	for idx, netIps := range n.IPs {
		_, netw, err := net.ParseCIDR(fmt.Sprintf("%s/%d", netIps.Address, netIps.Prefix))
		if err != nil {
			return xmlIdx, err
		}

		if netw.Contains(net.ParseIP(ip)) {
			xmlIdx = idx
			break
		}
	}

	return xmlIdx, nil
}

func updateHost(virConn *libvirt.Libvirt, n libvirt.Network, ip, mac, name string, xmlIdx int) error {
	xmlDesc := getHostXMLDesc(ip, mac, name)
	log.Printf("Updating host with XML:\n%s", xmlDesc)
	// From https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkUpdateFlags
	// Update live and config for hosts to make update permanent across reboots
	return virConn.NetworkUpdateCompat(n, libvirt.NetworkUpdateCommandModify,
		libvirt.NetworkSectionIPDhcpHost, int32(xmlIdx), xmlDesc,
		libvirt.NetworkUpdateAffectConfig|libvirt.NetworkUpdateAffectLive)
}

func getHostXMLDesc(ip, mac, name string) string {
	dd := libvirtxml.NetworkDHCPHost{
		IP:   ip,
		MAC:  mac,
		Name: name,
	}
	tmp := struct {
		XMLName xml.Name `xml:"host"`
		libvirtxml.NetworkDHCPHost
	}{xml.Name{}, dd}
	xml, err := xmlMarshallIndented(tmp)
	if err != nil {
		panic("could not marshall host")
	}
	return xml
}

func xmlMarshallIndented(b interface{}) (string, error) {
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	enc.Indent("  ", "    ")
	if err := enc.Encode(b); err != nil {
		return "", fmt.Errorf("could not marshall this:\n%s", spew.Sdump(b))
	}
	return buf.String(), nil
}

func isError(err error, errorCode libvirt.ErrorNumber) bool {
	var perr libvirt.Error
	if errors.As(err, &perr) {
		return perr.Code == uint32(errorCode)
	}
	return false
}

func addHost(virConn *libvirt.Libvirt, n libvirt.Network, ip, mac, name string, xmlIdx int) error {
	xmlDesc := getHostXMLDesc(ip, mac, name)
	log.Printf("Adding host with XML:\n%s", xmlDesc)
	// From https://libvirt.org/html/libvirt-libvirt-network.html#virNetworkUpdateFlags
	// Update live and config for hosts to make update permanent across reboots
	return virConn.NetworkUpdateCompat(n, libvirt.NetworkUpdateCommandAddLast,
		libvirt.NetworkSectionIPDhcpHost, int32(xmlIdx), xmlDesc,
		libvirt.NetworkUpdateAffectConfig|libvirt.NetworkUpdateAffectLive)
}
