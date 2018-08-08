# vmmgt
A tool to manage virtual machines, including create/delete/list tools.

## prerequisites
libvirt-devel virt-install

## install
go get https://github.com/secawa/vmmgt.git

## help
vmmgt -h

## create
vmmgt -name new create -cpu 12 -memory 4096 -disk 50

## list
vmmgt list -v

## delete
vmmgt -n new delete
