# Table of Contents
- [Introduction](#introduction)
- [Installation](#installation)
- [License](#license)

# Introduction
introduction

# Installation
## Linux
linux installation
```azureuser@myVM:~$ sudo wget https://github.com/curltech/go-colla-node/releases/download/v0.0.1/go-colla-node-linux-0.0.1.tar.gz
azureuser@myVM:~$ sudo tar zxvf go-colla-node-linux-0.0.1.tar.gz
azureuser@myVM:~$ cd go-colla-node
azureuser@myVM:~/go-colla-node$ vi conf/peer1.yml
azureuser@myVM:~/go-colla-node$ ./main (or ./main -appname peer3)
azureuser@myVM:~/go-colla-node$ sqlite3 peer1.db
sqlite> select discoveryaddress from blc_myselfpeer;
/dns4/peer3.curltech.cc/tcp/5720/wss/p2p/12D3KooWNwQ9pHZzmfjk8rZp4a1k5gXhibKxZMBdkdg1mTJEAYse```

## Windows
windows installation

# License
Copyright 2020-2021 CURL TECH PTE. LTD.

Licensed under the AGPLv3: https://www.gnu.org/licenses/agpl-3.0.html
