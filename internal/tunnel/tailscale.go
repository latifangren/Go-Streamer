package tunnel

import (
	"context"
	"encoding/json"
	"net"
	"os/exec"
	"strings"
)

type tailscaleStatusJSON struct {
	BackendState string `json:"BackendState"`
	Self         struct {
		TailscaleIPs []string `json:"TailscaleIPs"`
	} `json:"Self"`
	TailscaleIPs []string `json:"TailscaleIPs"`
}

// TailscaleDetector mendeteksi instalasi, status daemon, dan IP Tailscale pada perangkat.
type TailscaleDetector struct {
	tailscaleBin string
}

// NewTailscaleDetector membuat instans baru TailscaleDetector.
func NewTailscaleDetector() *TailscaleDetector {
	bin, _ := exec.LookPath("tailscale")
	return &TailscaleDetector{tailscaleBin: bin}
}

// GetStatus memeriksa ketersediaan Tailscale, status koneksi aktif, dan alamat IP CGNAT (100.x.y.z).
func (d *TailscaleDetector) GetStatus(ctx context.Context) (installed bool, active bool, ip string, err error) {
	bin := d.tailscaleBin
	if bin == "" {
		if path, err := exec.LookPath("tailscale"); err == nil {
			bin = path
			d.tailscaleBin = path
		}
	}

	if bin != "" {
		installed = true
		cmd := exec.CommandContext(ctx, bin, "status", "--json")
		if out, err := cmd.Output(); err == nil {
			var st tailscaleStatusJSON
			if err := json.Unmarshal(out, &st); err == nil {
				if strings.EqualFold(st.BackendState, "Running") {
					active = true
				}

				// Cari IP IPv4 100.x.y.z dari Self.TailscaleIPs
				candidateIPs := append(st.Self.TailscaleIPs, st.TailscaleIPs...)
				for _, tip := range candidateIPs {
					if strings.HasPrefix(tip, "100.") {
						ip = tip
						break
					}
				}

				if active && ip != "" {
					return true, true, ip, nil
				}
			}
		}
	}

	// Fallback: periksa interface jaringan lokal (misal adapter tailscale0 atau VPN IP 100.64.0.0/10)
	if ifaceIP := findTailscaleInterfaceIP(); ifaceIP != "" {
		return true, true, ifaceIP, nil
	}

	return installed, active, ip, nil
}

// findTailscaleInterfaceIP mencari IP Tailscale langsung dari daftar network interface sistem operasi.
func findTailscaleInterfaceIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		name := strings.ToLower(iface.Name)
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}

			ip4 := ipNet.IP.To4()
			if ip4 == nil {
				continue
			}

			// Range CGNAT Tailscale: 100.64.0.0 s/d 100.127.255.255
			if ip4[0] == 100 && (ip4[1] >= 64 && ip4[1] <= 127) {
				return ip4.String()
			}

			if strings.Contains(name, "tailscale") {
				return ip4.String()
			}
		}
	}

	return ""
}
