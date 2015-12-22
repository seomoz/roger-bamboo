# -*- mode: ruby -*-
# vi: set ft=ruby :

nodes = [
  { :hostname => 'junction-builder.local', :ip => '192.168.100.100', :mem => 1024, :cpus => 1 }
]

Vagrant.configure(2) do |config|
  config.vm.box = "ubuntu/trusty64"

  config.vm.provision "shell", path: "vagrant-provision.sh"

  # The following lines were added to support access when connected via vpn (see: http://akrabat.com/sharing-host-vpn-with-vagrant/)
  config.vm.provider :virtualbox do |vb|
    vb.customize ["modifyvm", :id, "--natdnshostresolver1", "on"]
  end

  nodes.each do |entry|
    config.vm.define entry[:hostname] do |node|
      node.vm.hostname = entry[:hostname]
      node.vm.network :private_network, ip: entry[:ip]
      node.vm.provider :virtualbox do |vb|
        vb.memory = entry[:mem]
        vb.cpus = entry[:cpus]
      end
    end
  end
end
