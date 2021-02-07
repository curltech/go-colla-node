package server

import (
	"context"
	"encoding/base64"
	"fmt"
	collaconfig "github.com/curltech/go-colla-core/config"
	"github.com/curltech/go-colla-core/logger"
	"github.com/curltech/go-colla-node/libp2p/global"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/plugin/loader" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/peer"
)

type ipfsPeer struct {
	Context  context.Context
	RepoPath string
	IpfsNode *core.IpfsNode
	CoreAPI  icore.CoreAPI
}

var IpfsPeer *ipfsPeer

//	安装插件
func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		logger.Sugar.Errorf("error loading plugins: %s", err)
		panic(err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		logger.Sugar.Errorf("error initializing plugins: %s", err)
		panic(err)
	}

	if err := plugins.Inject(); err != nil {
		logger.Sugar.Errorf("error initializing plugins: %s", err)
		panic(err)
	}

	return nil
}

/**
创建新的身份
*/
func NewIdentity() (*config.Identity, error) {
	identity, err := config.CreateIdentity(ioutil.Discard, []options.KeyGenerateOption{options.Key.Type(options.Ed25519Key)})
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		panic(err)
	}
	return &identity, nil
}

/**
获取libp2p的身份
*/
func identity() (*config.Identity, error) {
	buf, err := global.Global.PeerPrivateKey.Bytes()
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return nil, err
	}
	id := config.Identity{PeerID: string(global.Global.PeerId), PrivKey: base64.StdEncoding.EncodeToString(buf)}

	return &id, nil
}

//创建文件区域
func createRepo() error {
	id, err := identity()
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}
	cfg, err := config.InitWithIdentity(*id)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}

	IpfsPeer.RepoPath = collaconfig.IpfsParams.RepoPath
	err = fsrepo.Init(IpfsPeer.RepoPath, cfg)
	if err != nil {
		logger.Sugar.Errorf("failed to init ephemeral node: %s", err)
		return err
	}

	return nil
}

// 创建ipfs节点
func createNode() error {
	repo, err := fsrepo.Open(IpfsPeer.RepoPath)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}

	// Construct the node
	// 可以在此配置ipfs的参数并和libp2p节点融合或者独立
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTServerOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		//Host: func(ctx context.Context, id peer.ID, ps peerstore.Peerstore, options ...libp2p1.Option) (host.Host, error) {
		//	return global.Global.Host, nil
		//},
		Repo: repo,
	}

	IpfsPeer.IpfsNode, err = core.NewNode(IpfsPeer.Context, nodeOptions)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}

	// Attach the Core API to the constructed node
	IpfsPeer.CoreAPI, err = coreapi.NewCoreAPI(IpfsPeer.IpfsNode)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}

	return nil
}

// 创建节点，文件区域的设置从环境变量读取
func spawnDefault() error {
	var err error
	IpfsPeer.RepoPath, err = config.PathRoot()
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}

	if err := setupPlugins(IpfsPeer.RepoPath); err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}

	return createNode()
}

// 创建节点，文件区域的设置从配置文件读取
func spawnEphemeral() error {
	if err := setupPlugins(collaconfig.IpfsParams.ExternalPluginsPath); err != nil {
		logger.Sugar.Errorf("%v", err)
		return err
	}

	err := createRepo()
	if err != nil {
		logger.Sugar.Errorf("failed to create temp repo: %s", err)
		return err
	}

	return createNode()
}

//连接其他peers
func connectToPeers(peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peer.AddrInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		//pii, err := peerstore.InfoFromP2pAddr(addr)
		pii, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peer.AddrInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peerstore.PeerInfo) {
			defer wg.Done()
			err := IpfsPeer.CoreAPI.Swarm().Connect(IpfsPeer.Context, *peerInfo)
			if err != nil {
				logger.Sugar.Errorf("failed to connect to %s: %s", peerInfo.ID, err)
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}

/**
根据路径获取文件的句柄
*/
func getFile(path string) (files.File, *os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	st, err := file.Stat()
	if err != nil {
		return nil, file, err
	}

	f, err := files.NewReaderPathFile(path, file, st)
	if err != nil {
		return nil, file, err
	}

	return f, file, nil
}

/**
根据文件路径返回节点下的文件
*/
func getFileNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		logger.Sugar.Errorf("%v", err)
		return nil, err
	}

	return f, nil
}

/**
启动ipfs的节点
*/
func Start() {
	/// --- Part I: Getting a IPFS node running
	logger.Sugar.Infof("-- Getting an IPFS node running -- ")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Spawn a node using a temporary path, creating a temporary repo for the run
	logger.Sugar.Infof("Spawning node on a temporary repo")
	var err error
	IpfsPeer = &ipfsPeer{Context: ctx}
	err = spawnEphemeral()
	if err != nil {
		logger.Sugar.Errorf("failed to spawn ephemeral node: %s", err)
		return
	}

	logger.Sugar.Infof("successfully start ipfs node:%v in %v", IpfsPeer.IpfsNode.PeerHost.ID(), IpfsPeer.IpfsNode.PeerHost.Addrs())
	logger.Sugar.Infof("in repo path:%v, enjoy it!", IpfsPeer.RepoPath)

	select {}
}

func AddFile(filename string) string {
	//加一个文件和路径到ipfs
	/// --- Part II: Adding a file and a directory to IPFS
	logger.Sugar.Infof("\n-- Adding and getting back files & directories --")
	someFile, file, err := getFile(filename)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		logger.Sugar.Errorf("Could not get File: %s", err)

		return ""
	}
	defer someFile.Close()
	cidFile, err := IpfsPeer.CoreAPI.Unixfs().Add(IpfsPeer.Context, someFile)
	if err != nil {
		logger.Sugar.Errorf("Could not add File: %s", err)

		return ""
	}

	logger.Sugar.Infof("Added file to IPFS with CID %s\n", cidFile.String())

	return cidFile.String()
}

func AddDirectory(path string) string {
	someDirectory, err := getFileNode(path)
	if err != nil {
		logger.Sugar.Errorf("Could not get File: %s", err)

		return ""
	}

	cidDirectory, err := IpfsPeer.CoreAPI.Unixfs().Add(IpfsPeer.Context, someDirectory)
	if err != nil {
		logger.Sugar.Errorf("Could not add Directory: %s", err)

		return ""
	}

	logger.Sugar.Infof("Added directory to IPFS with CID %s\n", cidDirectory.String())

	return cidDirectory.String()
}

func GetFile(cid string, path string) (string, error) {
	cidFile := icorepath.New(cid)
	var outputPathFile string
	id := strings.Split(cidFile.String(), "/")[2]
	if strings.HasSuffix(path, "/") == false {
		outputPathFile = path + "/" + id
	} else {
		outputPathFile = path + id
	}

	rootNodeFile, err := IpfsPeer.CoreAPI.Unixfs().Get(IpfsPeer.Context, cidFile)
	if err != nil {
		logger.Sugar.Errorf("Could not get file with CID: %s", err)

		return outputPathFile, err
	}

	err = files.WriteTo(rootNodeFile, outputPathFile)
	if err != nil {
		logger.Sugar.Errorf("Could not write out the fetched CID: %s", err)

		return outputPathFile, err
	}

	logger.Sugar.Infof("Got file back from IPFS (IPFS path: %s) and wrote it to %s\n", cidFile.String(), outputPathFile)

	return outputPathFile, nil
}

func GetDirectory(cid string, path string) (string, error) {
	cidDirectory := icorepath.New(cid)
	var outputPathDirectory string
	id := strings.Split(cidDirectory.String(), "/")[2]
	if strings.HasSuffix(path, "/") == false {
		outputPathDirectory = path + "/" + id
	} else {
		outputPathDirectory = path + id
	}
	rootNodeDirectory, err := IpfsPeer.CoreAPI.Unixfs().Get(IpfsPeer.Context, cidDirectory)
	if err != nil {
		logger.Sugar.Errorf("Could not get file with CID: %s", err)

		return outputPathDirectory, err
	}

	err = files.WriteTo(rootNodeDirectory, outputPathDirectory)
	if err != nil {
		logger.Sugar.Errorf("Could not write out the fetched CID: %s", err)

		return outputPathDirectory, err
	}

	logger.Sugar.Infof("Got directory back from IPFS (IPFS path: %s) and wrote it to %s\n", cidDirectory.String(), outputPathDirectory)

	return outputPathDirectory, nil
}

func Connect() {
	fmt.Println("\n-- Going to connect to a few nodes in the Network as bootstrappers --")
	bootstrapNodes := collaconfig.IpfsParams.BootstrapNodes
	go connectToPeers(bootstrapNodes)
}
