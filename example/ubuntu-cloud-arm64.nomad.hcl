job "qemu-vm" {
  datacenters = ["dc1"]

  group "qemu-vm" {

    task "vm" {

      artifact {
        source      = "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-arm64.img"
        destination = "local/jammy-server-cloudimg-arm64.img"
        mode        = "file"
      }

      driver = "libvirt"

      config {

        type = "qemu"

        os {
          arch    = "aarch64"
          machine = "virt"
          type    = "hvm"
        }

        network_interface {
          address      = "192.168.122.13"
          network_name = "default"
        }

        disk {
          source = "${NOMAD_TASK_DIR}/jammy-server-cloudimg-arm64.img"
          target = "hda"
          device = "disk"

          driver {
            name = "qemu"
            type = "qcow2"
          }
        }
      }
    }
  }
}
