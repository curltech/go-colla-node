# Table of Contents
- [Introduction](#introduction)
- [Installation](#installation)
- [License](#license)

# Introduction
Colla Network is a decentralized peer-to-peer communication and storage network, Colla App (e.g. CollaChat) connects to Colla Network via selected Colla Node, there is no centralized storage and control of user data, user can specify which node or even easily setup own node(s) to connect to.

# Installation
Get ready your server, domain and certificate first. You may rent a cloud server from CSP (or try on from a free instance), buy a domain or get a free one, use [ACME](https://github.com/acmesh-official/acme.sh) to automatically issue & renew the free certificates from Let's Encrypt.

## Tested OS
No  | OS  | Version and Spec.
 ---- | ----- | ------  
 1  | Ubuntu | 16.04/18.04/20.04, minimal(1vCPUs + 1GiB)
 2  | Windows | Windows Server 2016

## Linux
1. Download and uncompress
```
azureuser@myVM:~$ sudo wget https://github.com/curltech/go-colla-node/releases/download/v0.0.1/go-colla-node-linux-0.0.1.tar.gz
azureuser@myVM:~$ sudo tar zxvf go-colla-node-linux-0.0.1.tar.gz
```

2. Configure and run
```
azureuser@myVM:~$ cd go-colla-node
azureuser@myVM:~/go-colla-node$ vi conf/peer1.yml
azureuser@myVM:~/go-colla-node$ ./main (or ./main -appname peer1)
```

3. Use custom node in Colla App (e.g. CollaChat)
```
azureuser@myVM:~/go-colla-node$ sqlite3 peer1.db
sqlite> select discoveryaddress from blc_myselfpeer;
/dns4/peer1.curltech.cc/tcp/5720/wss/p2p/12D3KooWNwQ9pHZzmfjk8rZp4a1k5gXhibKxZMBdkdg1mTJEAYse
```

![login](https://github.com/curltech/go-colla-node/blob/main/readmeImg/customNode-login.png)

![accountInformation](https://github.com/curltech/go-colla-node/blob/main/readmeImg/customNode-accountInformation.png)

## Windows
windows installation

# License
Copyright 2020-2021 CURL TECH PTE. LTD.

Licensed under the AGPLv3: https://www.gnu.org/licenses/agpl-3.0.html
