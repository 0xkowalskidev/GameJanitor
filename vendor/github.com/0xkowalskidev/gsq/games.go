package gsq

type GameConfig struct {
	Slug             string
	Name             string
	Aliases          []string
	AppID            uint32
	DefaultGamePort  uint16
	DefaultQueryPort uint16
	Protocol         string
	EOSClientID      string
	EOSClientSecret  string
	EOSDeploymentID  string
	EOSExternalAuth  bool
	EOSWildcard      bool
	EOSAttributes    map[string]string
}

var gameRegistry = []GameConfig{
	// Minecraft
	{Slug: "minecraft", Name: "Minecraft", Aliases: []string{"mc"}, DefaultGamePort: 25565, DefaultQueryPort: 25565, Protocol: "minecraft"},
	{Slug: "minecraft-bedrock", Name: "Minecraft: Bedrock Edition", Aliases: []string{"mcbe", "bedrock"}, DefaultGamePort: 19132, DefaultQueryPort: 19132, Protocol: "raknet"},

	// EOS (Epic Online Services)
	// ClientID and ClientSecret are public credentials shipped in each game's binary — they grant
	// read-only access to Epic's matchmaking API. They may rotate when a game updates, so users
	// can override them with --eos-client-id/--eos-client-secret. DeploymentID is tied to the game
	// itself (not the client role) and should not rotate.
	{
		Slug: "ark-survival-ascended", Name: "ARK: Survival Ascended", Aliases: []string{"asa"},
		DefaultGamePort: 7777, DefaultQueryPort: 7777, Protocol: "eos",
		EOSClientID: "xyza7891muomRmynIIHaJB9COBKkwj6n", EOSClientSecret: "PP5UGxysEieNfSrEicaD1N2Bb3TdXuD7xHYcsdUHZ7s",
		EOSDeploymentID: "ad9a8feffb3b4b2ca315546f038c3ae2", EOSWildcard: true,
		EOSAttributes: map[string]string{"name": "CUSTOMSERVERNAME_s", "map": "MAPNAME_s", "password": "SERVERPASSWORD_b", "version": "BUILDID_s"},
	},
	{
		Slug: "squad", Name: "Squad", Aliases: []string{},
		DefaultGamePort: 7787, DefaultQueryPort: 7787, Protocol: "eos",
		EOSClientID: "xyza7891J7d3GU8ZIwCoC5xdBsdoqVWA", EOSClientSecret: "4SLVBqAm09q776SIlQRTD6moM/bnGAWhDSqOxJAIS0s",
		EOSDeploymentID: "5dee4062a90b42cd98fcad618b6636c2", EOSExternalAuth: true,
		EOSAttributes: map[string]string{"name": "SERVERNAME_s", "password": "PASSWORD_b", "version": "GAMEVERSION_s"},
	},
	{
		Slug: "palworld", Name: "Palworld", Aliases: []string{"pw"},
		DefaultGamePort: 8211, DefaultQueryPort: 8211, Protocol: "eos",
		EOSClientID: "xyza78916PZ5DF0fAahu4tnrKKyFpqRE", EOSClientSecret: "j0NapLEPm3R3EOrlQiM8cRLKq3Rt02ZVVwT0SkZstSg",
		EOSDeploymentID: "0a18471f93d448e2a1f60e47e03d3413", EOSExternalAuth: true,
		EOSAttributes: map[string]string{"name": "NAME_s", "map": "MAPNAME_s", "password": "ISPASSWORD_b", "version": "VERSION_s"},
	},
	{
		Slug: "the-isle-evrima", Name: "The Isle: EVRIMA", Aliases: []string{"evrima"},
		DefaultGamePort: 7777, DefaultQueryPort: 7777, Protocol: "eos",
		EOSClientID: "xyza7891gk5PRo3J7G9puCJGFJjmEguW", EOSClientSecret: "pKWl6t5i9NJK8gTpVlAxzENZ65P8hYzodV8Dqe5Rlc8",
		EOSDeploymentID: "6db6bea492f94b1bbdfcdfe3e4f898dc",
		EOSAttributes:   map[string]string{"name": "SERVERNAME_s", "map": "MAP_NAME_s", "version": "SERVER_VERSION_s"},
	},

	// Source engine (A2S)
	{Slug: "the-isle", Name: "The Isle", Aliases: []string{"isle"}, AppID: 376210, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "counter-strike-2", Name: "Counter-Strike 2", Aliases: []string{"cs2"}, AppID: 730, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "counter-strike", Name: "Counter-Strike 1.6", Aliases: []string{"cs16", "cs"}, AppID: 10, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "counter-strike-source", Name: "Counter-Strike: Source", Aliases: []string{"css"}, AppID: 240, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "counter-strike-go", Name: "Counter-Strike: GO", Aliases: []string{"csgo"}, AppID: 730, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "rust", Name: "Rust", Aliases: []string{}, AppID: 252490, DefaultGamePort: 28015, DefaultQueryPort: 28017, Protocol: "source"},
	{Slug: "ark-survival-evolved", Name: "ARK: Survival Evolved", Aliases: []string{"ark", "ase"}, AppID: 346110, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "valheim", Name: "Valheim", Aliases: []string{}, AppID: 892970, DefaultGamePort: 2456, DefaultQueryPort: 2457, Protocol: "source"},
	{Slug: "dayz", Name: "DayZ", Aliases: []string{}, AppID: 221100, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source"},
	{Slug: "garrys-mod", Name: "Garry's Mod", Aliases: []string{"gmod"}, AppID: 4000, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "team-fortress-2", Name: "Team Fortress 2", Aliases: []string{"tf2"}, AppID: 440, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "conan-exiles", Name: "Conan Exiles", Aliases: []string{"conan"}, AppID: 440900, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "7-days-to-die", Name: "7 Days to Die", Aliases: []string{"7dtd"}, AppID: 251570, DefaultGamePort: 26900, DefaultQueryPort: 26900, Protocol: "source"},
	{Slug: "hell-let-loose", Name: "Hell Let Loose", Aliases: []string{"hll"}, AppID: 686810, DefaultGamePort: 7777, DefaultQueryPort: 7778, Protocol: "source"},
	{Slug: "enshrouded", Name: "Enshrouded", Aliases: []string{}, AppID: 1203620, DefaultGamePort: 15636, DefaultQueryPort: 15637, Protocol: "source"},
	{Slug: "v-rising", Name: "V Rising", Aliases: []string{"vrising"}, AppID: 1604030, DefaultGamePort: 9876, DefaultQueryPort: 9877, Protocol: "source"},
	{Slug: "unturned", Name: "Unturned", Aliases: []string{}, AppID: 304930, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "project-zomboid", Name: "Project Zomboid", Aliases: []string{"pz"}, AppID: 108600, DefaultGamePort: 16261, DefaultQueryPort: 16261, Protocol: "source"},
	{Slug: "space-engineers", Name: "Space Engineers", Aliases: []string{"se"}, AppID: 244850, DefaultGamePort: 27016, DefaultQueryPort: 27016, Protocol: "source"},
	{Slug: "insurgency-sandstorm", Name: "Insurgency: Sandstorm", Aliases: []string{"inss"}, AppID: 581320, DefaultGamePort: 27102, DefaultQueryPort: 27131, Protocol: "source"},
	{Slug: "insurgency", Name: "Insurgency", Aliases: []string{"ins"}, AppID: 222880, DefaultGamePort: 27015, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "mordhau", Name: "Mordhau", Aliases: []string{}, AppID: 629760, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "killing-floor-2", Name: "Killing Floor 2", Aliases: []string{"kf2"}, AppID: 232090, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "barotrauma", Name: "Barotrauma", Aliases: []string{}, AppID: 602960, DefaultGamePort: 27015, DefaultQueryPort: 27016, Protocol: "source"},
	{Slug: "arma-3", Name: "Arma 3", Aliases: []string{"arma3"}, AppID: 107410, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source"},
	{Slug: "arma-2", Name: "Arma 2", Aliases: []string{"arma2"}, AppID: 33900, DefaultGamePort: 2302, DefaultQueryPort: 2303, Protocol: "source"},
	{Slug: "squad-44", Name: "Squad 44", Aliases: []string{"s44"}, AppID: 736220, DefaultGamePort: 7787, DefaultQueryPort: 27165, Protocol: "source"},
	{Slug: "beyond-the-wire", Name: "Beyond the Wire", Aliases: []string{"btw"}, AppID: 1058650, DefaultGamePort: 7787, DefaultQueryPort: 27165, Protocol: "source"},
	{Slug: "rising-storm-2", Name: "Rising Storm 2: Vietnam", Aliases: []string{"rs2"}, AppID: 418460, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "the-front", Name: "The Front", Aliases: []string{}, AppID: 2285150, DefaultGamePort: 5001, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "soulmask", Name: "Soulmask", Aliases: []string{}, AppID: 2646460, DefaultGamePort: 8777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "myth-of-empires", Name: "Myth of Empires", Aliases: []string{"moe"}, AppID: 1371580, DefaultGamePort: 11888, DefaultQueryPort: 12888, Protocol: "source"},
	{Slug: "atlas", Name: "Atlas", Aliases: []string{}, AppID: 834910, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "dark-and-light", Name: "Dark and Light", Aliases: []string{"dnl"}, AppID: 529180, DefaultGamePort: 7777, DefaultQueryPort: 27015, Protocol: "source"},
	{Slug: "pixark", Name: "PixARK", Aliases: []string{}, AppID: 593600, DefaultGamePort: 27015, DefaultQueryPort: 27016, Protocol: "source"},
	{Slug: "battalion-1944", Name: "Battalion 1944", Aliases: []string{"battalion"}, AppID: 489940, DefaultGamePort: 7777, DefaultQueryPort: 7780, Protocol: "source"},
	{Slug: "starbound", Name: "Starbound", Aliases: []string{}, AppID: 211820, DefaultGamePort: 21025, DefaultQueryPort: 21025, Protocol: "source"},
}

// SupportedGames returns all registered game configs.
func SupportedGames() []GameConfig {
	result := make([]GameConfig, len(gameRegistry))
	copy(result, gameRegistry)
	return result
}

// LookupGameByAppID finds a game by Steam AppID. Returns nil if not found.
func LookupGameByAppID(appID uint32) *GameConfig {
	if appID == 0 {
		return nil
	}
	for i := range gameRegistry {
		if gameRegistry[i].AppID == appID {
			return &gameRegistry[i]
		}
	}
	return nil
}

// LookupGame finds a game by slug or alias. Returns nil if not found.
func LookupGame(name string) *GameConfig {
	for i := range gameRegistry {
		g := &gameRegistry[i]
		if g.Slug == name {
			return g
		}
		for _, alias := range g.Aliases {
			if alias == name {
				return g
			}
		}
	}
	return nil
}
