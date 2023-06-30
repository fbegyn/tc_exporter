# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "debian/bookworm64"
  config.vm.network "forwarded_port", guest: 9704, host: 9704

  config.vm.provision "shell", inline: <<-SHELL
    echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
    apt-get update
    apt-get install -y golang make git nfpm
  SHELL
end
