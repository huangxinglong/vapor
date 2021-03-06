package config

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/common"
)

var (
	// CommonConfig means config object
	CommonConfig *Config
)

type Config struct {
	// Top level options use an anonymous struct
	BaseConfig `mapstructure:",squash"`
	// Options for services
	P2P       *P2PConfig          `mapstructure:"p2p"`
	Wallet    *WalletConfig       `mapstructure:"wallet"`
	Auth      *RPCAuthConfig      `mapstructure:"auth"`
	Web       *WebConfig          `mapstructure:"web"`
	Side      *SideChainConfig    `mapstructure:"side"`
	MainChain *MainChainRpcConfig `mapstructure:"mainchain"`
	Websocket *WebsocketConfig    `mapstructure:"ws"`
	Consensus *ConsensusConfig    `mapstructure:"consensus"`
}

// Default configurable parameters.
func DefaultConfig() *Config {
	return &Config{
		BaseConfig: DefaultBaseConfig(),
		P2P:        DefaultP2PConfig(),
		Wallet:     DefaultWalletConfig(),
		Auth:       DefaultRPCAuthConfig(),
		Web:        DefaultWebConfig(),
		Side:       DefaultSideChainConfig(),
		MainChain:  DefaultMainChainRpc(),
		Websocket:  DefaultWebsocketConfig(),
		Consensus:  DefaultConsensusCOnfig(),
	}
}

// Set the RootDir for all Config structs
func (cfg *Config) SetRoot(root string) *Config {
	cfg.BaseConfig.RootDir = root
	return cfg
}

//-----------------------------------------------------------------------------
// BaseConfig
type BaseConfig struct {
	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir string `mapstructure:"home"`

	//The ID of the network to json
	ChainID string `mapstructure:"chain_id"`

	//log level to set
	LogLevel string `mapstructure:"log_level"`

	// A custom human readable name for this node
	Moniker string `mapstructure:"moniker"`

	// TCP or UNIX socket address for the profiling server to listen on
	ProfListenAddress string `mapstructure:"prof_laddr"`

	Mining bool `mapstructure:"mining"`

	// Database backend: leveldb | memdb
	DBBackend string `mapstructure:"db_backend"`

	// Database directory
	DBPath string `mapstructure:"db_dir"`

	// Keystore directory
	KeysPath string `mapstructure:"keys_dir"`

	ApiAddress string `mapstructure:"api_addr"`

	VaultMode bool `mapstructure:"vault_mode"`

	// log file name
	LogFile string `mapstructure:"log_file"`

	// Validate pegin proof by checking bytom transaction inclusion in mainchain.
	ValidatePegin bool   `mapstructure:"validate_pegin"`
	Signer        string `mapstructure:"signer"`

	ConsensusConfigFile string `mapstructure:"consensus_config_file"`

	IpfsAddress string `mapstructure:"ipfs_addr"`
}

// Default configurable base parameters.
func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		Moniker:           "anonymous",
		ProfListenAddress: "",
		Mining:            false,
		DBBackend:         "leveldb",
		DBPath:            "data",
		KeysPath:          "keystore",
		IpfsAddress:       "127.0.0.1:5001",
	}
}

func (b BaseConfig) DBDir() string {
	return rootify(b.DBPath, b.RootDir)
}

func (b BaseConfig) KeysDir() string {
	return rootify(b.KeysPath, b.RootDir)
}

// P2PConfig
type P2PConfig struct {
	ListenAddress    string `mapstructure:"laddr"`
	Seeds            string `mapstructure:"seeds"`
	SkipUPNP         bool   `mapstructure:"skip_upnp"`
	MaxNumPeers      int    `mapstructure:"max_num_peers"`
	HandshakeTimeout int    `mapstructure:"handshake_timeout"`
	DialTimeout      int    `mapstructure:"dial_timeout"`
	ProxyAddress     string `mapstructure:"proxy_address"`
	ProxyUsername    string `mapstructure:"proxy_username"`
	ProxyPassword    string `mapstructure:"proxy_password"`
}

// Default configurable p2p parameters.
func DefaultP2PConfig() *P2PConfig {
	return &P2PConfig{
		ListenAddress:    "tcp://0.0.0.0:46656",
		SkipUPNP:         false,
		MaxNumPeers:      50,
		HandshakeTimeout: 30,
		DialTimeout:      3,
		ProxyAddress:     "",
		ProxyUsername:    "",
		ProxyPassword:    "",
	}
}

//-----------------------------------------------------------------------------
type WalletConfig struct {
	Disable  bool   `mapstructure:"disable"`
	Rescan   bool   `mapstructure:"rescan"`
	MaxTxFee uint64 `mapstructure:"max_tx_fee"`
}

type RPCAuthConfig struct {
	Disable bool `mapstructure:"disable"`
}

type WebConfig struct {
	Closed bool `mapstructure:"closed"`
}

type SideChainConfig struct {
	FedpegXPubs            string `mapstructure:"fedpeg_xpubs"`
	SignBlockXPubs         string `mapstructure:"sign_block_xpubs"`
	PeginMinDepth          uint64 `mapstructure:"pegin_confirmation_depth"`
	ParentGenesisBlockHash string `mapstructure:"parent_genesis_block_hash"`
}

type MainChainRpcConfig struct {
	MainchainRpcHost string `mapstructure:"mainchain_rpc_host"`
	MainchainRpcPort string `mapstructure:"mainchain_rpc_port"`
	MainchainToken   string `mapstructure:"mainchain_rpc_token"`
}

type WebsocketConfig struct {
	MaxNumWebsockets     int `mapstructure:"max_num_websockets"`
	MaxNumConcurrentReqs int `mapstructure:"max_num_concurrent_reqs"`
}

type ConsensusConfig struct {
	Type             string   `mapstructure:"consensus_type"`
	Period           uint64   `json:"period"`            // Number of seconds between blocks to enforce
	MaxSignerCount   uint64   `json:"max_signers_count"` // Max count of signers
	MinVoterBalance  uint64   `json:"min_boter_balance"` // Min voter balance to valid this vote
	GenesisTimestamp uint64   `json:"genesis_timestamp"` // The LoopStartTime of first Block
	Coinbase         string   `json:"coinbase"`
	XPrv             string   `json:"xprv"`
	SelfVoteSigners  []string `json:"signers"` // Signers vote by themselves to seal the block, make sure the signer accounts are pre-funded
	Signers          []common.Address
}

type DposConfig struct {
	Period           uint64   `json:"period"`            // Number of seconds between blocks to enforce
	MaxSignerCount   uint64   `json:"max_signers_count"` // Max count of signers
	MinVoterBalance  uint64   `json:"min_boter_balance"` // Min voter balance to valid this vote
	GenesisTimestamp uint64   `json:"genesis_timestamp"` // The LoopStartTime of first Block
	Coinbase         string   `json:"coinbase"`
	XPrv             string   `json:"xprv"`
	SelfVoteSigners  []string `json:"signers"` // Signers vote by themselves to seal the block, make sure the signer accounts are pre-funded
	Signers          []common.Address
}

// Default configurable rpc's auth parameters.
func DefaultRPCAuthConfig() *RPCAuthConfig {
	return &RPCAuthConfig{
		Disable: false,
	}
}

// Default configurable web parameters.
func DefaultWebConfig() *WebConfig {
	return &WebConfig{
		Closed: false,
	}
}

// Default configurable wallet parameters.
func DefaultWalletConfig() *WalletConfig {
	return &WalletConfig{
		Disable:  false,
		Rescan:   false,
		MaxTxFee: uint64(1000000000),
	}
}

// DeafultSideChainConfig for sidechain
func DefaultSideChainConfig() *SideChainConfig {
	return &SideChainConfig{
		PeginMinDepth:          6,
		ParentGenesisBlockHash: "a75483474799ea1aa6bb910a1a5025b4372bf20bef20f246a2c2dc5e12e8a053",
	}
}

func DefaultMainChainRpc() *MainChainRpcConfig {
	return &MainChainRpcConfig{
		MainchainRpcHost: "127.0.0.1",
		MainchainRpcPort: "9888",
	}
}

func DefaultWebsocketConfig() *WebsocketConfig {
	return &WebsocketConfig{
		MaxNumWebsockets:     25,
		MaxNumConcurrentReqs: 20,
	}
}

func DefaultDposConfig() *DposConfig {
	return &DposConfig{
		Period:           1,
		MaxSignerCount:   1,
		MinVoterBalance:  0,
		GenesisTimestamp: 1524549600,
	}
}

func DefaultConsensusCOnfig() *ConsensusConfig {
	return &ConsensusConfig{
		Type:             "dpos",
		Period:           1,
		MaxSignerCount:   1,
		MinVoterBalance:  0,
		GenesisTimestamp: 1524549600}
}

//-----------------------------------------------------------------------------
// Utils

// helper function to make config creation independent of root dir
func rootify(path, root string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home == "" {
		return "./.bytom_sidechain"
	}
	switch runtime.GOOS {
	case "darwin":
		// In order to be compatible with old data path,
		// copy the data from the old path to the new path
		oldPath := filepath.Join(home, "Library", "Bytom_sidechain")
		newPath := filepath.Join(home, "Library", "Application Support", "Bytom_sidechain")
		if !isFolderNotExists(oldPath) && isFolderNotExists(newPath) {
			if err := os.Rename(oldPath, newPath); err != nil {
				log.Errorf("DefaultDataDir: %v", err)
				return oldPath
			}
		}
		return newPath
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Bytom_sidechain")
	default:
		return filepath.Join(home, ".bytom_sidechain")
	}
}

func isFolderNotExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
