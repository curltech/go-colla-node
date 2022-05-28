### The backbone of the Colla Network - an unstoppable Web3 decentralized peer-to-peer communication and storage network.

[![use](https://img.shields.io/badge/use-go--libp2p-yellowgreen.svg)](https://github.com/libp2p/go-libp2p)
[![use](https://img.shields.io/badge/use-go--ipfs-blue.svg)](https://github.com/ipfs/go-ipfs)
[![use](https://img.shields.io/badge/use-pion-red.svg)](https://github.com/pion)

# Table of Contents
- [Introduction](#introduction)
- [Build](#Build)
- [Installation](#installation)
- [License](#license)

# Introduction
Colla Network is an unstoppable Web3 decentralized peer-to-peer communication and storage network, Colla DApp (e.g. [CollaChat](https://github.com/curltech/collachat)) connects to it via Colla Node. There is no centralized storage and control of user data, users can specify which Node(s) or even easily setup own Node(s) to connect to.

Learn more at https://curltech.io/#/collachat/.

# Build
## Optimized Settings
It's recommended to optimize following settings before building your own executable file:
No  | Package | File | Default Setting | Recommended Setting
 ---- | ----- | ------ | ----- | ------  
 1  | go-mplex@v0.7.0 | multiplex.go | MaxMessageSize = 1 << 20, BufferSize     = 4096 | MaxMessageSize = 1 << 30, BufferSize     = 1 << 20
 2  | go-libp2p-kad-dht@v0.15.0 | internal\net\massage_manager.go | var dhtReadMessageTimeout = 10 * time.Second | var dhtReadMessageTimeout = 300 * time.Second
 3  | go-libp2p-core@v0.15.1 | network\network.go | const MessageSizeMax = 1 << 22 | const MessageSizeMax = 1 << 30
 4  | go-msgio@v0.2.0 | msgio.go | defaultMaxSize = 8 * 1024 * 1024 | defaultMaxSize = 1024 * 1024 * 1024

# Installation
Get ready your server, domain and certificate first. You may rent a cloud server from CSP (or try on from a free instance), buy a domain or get a free one, use [ACME](https://github.com/acmesh-official/acme.sh) to automatically issue & renew the free certificates from Let's Encrypt.

## Tested OS
No  | OS  | Version and Spec.
 ---- | ----- | ------  
 1  | Ubuntu | 16.04/18.04/20.04, minimal 1vCPUs + 1GiB
 2  | Windows | Windows Server 2016

## Linux
1. Download and uncompress
```
azureuser@myVM:~$ sudo wget https://github.com/curltech/go-colla-node/releases/download/v0.0.3/go-colla-node-linux-0.0.3.tar.gz
azureuser@myVM:~$ sudo tar zxvf go-colla-node-linux-0.0.3.tar.gz
```

2. Configure and run
```
azureuser@myVM:~$ cd go-colla-node
azureuser@myVM:~/go-colla-node$ vi conf/peer1.yml
azureuser@myVM:~/go-colla-node$ ./main (or ./main -appname peer1)
```

3. Use custom node in Colla DApp (e.g. [CollaChat](https://github.com/curltech/collachat))
```
azureuser@myVM:~/go-colla-node$ sqlite3 peer1.db
sqlite> select discoveryaddress from blc_myselfpeer;
/dns4/peer1.curltech.cc/tcp/5720/wss/p2p/12D3KooWNwQ9pHZzmfjk8rZp4a1k5gXhibKxZMBdkdg1mTJEAYse
```

![login](https://github.com/curltech/go-colla-node/blob/main/readmeImg/customNode-login.png)

![accountInformation](https://github.com/curltech/go-colla-node/blob/main/readmeImg/customNode-accountInformation.png)

## Windows
Use zip file for Windows instead of tar.gz file for Linux, and follow the same steps as above.

# License
Copyright 2020-2021 CURL TECH PTE. LTD.

Licensed under the AGPLv3: https://www.gnu.org/licenses/agpl-3.0.html
