# vmmgt
A tool to manage virtual machines, including create/delete/list tools.

## prerequisites
libvirt-devel virt-install

## install
```
mkdir -p ~/go/src/10.0.24.42/wangdb/  
cd ~/go/src/10.0.24.42/wangdb/  
git clone http://10.0.24.42/wangdb/vmmgt.git  
cd vmmgt  
git submodule init  
git submodule update  
go build  
```

## help
./vmmgt -h

## create
./vmmgt create -cpu 12 -memory 4096 -disk 50 newname

## list
./vmmgt list -v

## delete
./vmmgt delete newname

## network
./vmmgt network list

## hostnetdev
./vmmgt hostnetdev list
