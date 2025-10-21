package main

type Config struct {
	SshPublicKeyPath  string `yaml:"ssh_public_key_path"`
	SshPrivateKeyPath string `yaml:"ssh_private_key_path"`

	InternalNetwork struct {
		Name   string `yaml:"name"`
		Subnet struct {
			Name        string `yaml:"name"`
			NetworkCidr string `yaml:"network_cidr"`
		} `yaml:"subnet"`
	} `yaml:"internal_network"`

	Bastion struct {
		VM vm     `yaml:",inline"`
		Ip string `yaml:"ip"`
	}

	NeutronRouter struct {
		Name            string `yaml:"name"`
		ExternalNetwork struct {
			Name string `yaml:"name"`
			ID   string `yaml:"id"`
		} `yaml:"external_network"`
		InternalNetwork struct {
			Name string `yaml:"name"`
		} `yaml:"internal_network"`
	} `yaml:"neutron_router"`

	K3s struct {
		Masters k3sInstance       `yaml:"masters"`
		Workers k3sInstance       `yaml:"workers"`
		Vars    map[string]string `yaml:"vars"`
		Ip      string            `yaml:"ip"`
	} `yaml:"k3s"`

	Firewall struct {
		FirewallExternal []FirewallRule `yaml:"firewall_external"`
		FirewallInternal []FirewallRule `yaml:"firewall_internal"`
	} `yaml:"firewall"`
}

type FirewallRule struct {
	Name         string   `yaml:"name"`
	Ports        []string `yaml:"ports"`
	SourceRanges []string `yaml:"source_ranges"`
}

type k3sInstance struct {
	VM     vm  `yaml:",inline"`
	Number int `yaml:"number"`
}

type vm struct {
	Name   string `yaml:"name"`
	Role   string `yaml:"role"`
	User   string `yaml:"user"`
	Image  string `yaml:"image"`
	Flavor string `yaml:"flavor"`
}

type TerraformOutput struct {
	K3sMasterPrivateIP struct {
		Value []string `json:"value"`
	} `json:"k3s_master_private_ip"`
	K3sWorkerPrivateIPs struct {
		Value []string `json:"value"`
	} `json:"k3s_worker_private_ip"`
	MasterFloatingIP struct {
		Value string `json:"value"`
	} `json:"master_floating_ip"`
}

type K3sCluster struct {
	Master []Node            `yaml:"masters"`
	Worker []Node            `yaml:"workers"`
	Vars   map[string]string `yaml:"vars"`
}

type Node struct {
	Ip string `yaml:"ip"`
}