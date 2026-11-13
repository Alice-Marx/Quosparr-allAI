package glm

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	// ErrNoCredentials means the store holds no usable credential.
	ErrNoCredentials = errors.New("glm: no credential")
	// ErrCredentialFormat means the store exists but could not be read.
	ErrCredentialFormat = errors.New("glm: credential store is unreadable")
)

// DataDir is the ZCode configuration directory below the home directory.
// The panel reads this machine's own store, so ZCODE_DATA_BASE_DIR is not
// consulted.
const DataDir = ".zcode"

// CredentialPath is the ZCode credential store of home. An empty home has no
// path, so nothing of the current directory is ever read.
func CredentialPath(home string) string {
	if home == "" {
		return ""
	}
	return filepath.Join(home, DataDir, "v2", "credentials.json")
}

// DatabasePath is the ZCode usage database of home.
func DatabasePath(home string) string {
	if home == "" {
		return ""
	}
	return filepath.Join(home, DataDir, "cli", "db", "db.sqlite")
}

// Region is the vendor site of a credential. The two sites keep separate
// accounts and separate credentials.
type Region int

const (
	// BigModel is the domestic site, bigmodel.cn.
	BigModel Region = iota
	// Zai is the international site, api.z.ai.
	Zai
)

// Host is the vendor host of the region, as the panel shows it.
func (r Region) Host() string {
	if r == Zai {
		return "api.z.ai"
	}
	return "bigmodel.cn"
}

// Kind is where a credential came from. The values are also the try order.
type Kind int

const (
	// KindNone is the zero value: no credential.
	KindNone Kind = iota
	// KindEnvKey is ZCODE_BIGMODEL_USAGE_API_KEY or BIGMODEL_USAGE_API_KEY.
	KindEnvKey
	// KindCodingPlan is a coding plan API key of the store.
	KindCodingPlan
	// KindOAuth is an OAuth access token of the store.
	KindOAuth
)

// Candidate is one credential the quota endpoint may accept. Its String
// omits the key on purpose, so a stray log line cannot leak it.
type Candidate struct {
	Key          string
	Kind         Kind
	Plan         string
	Region       Region
	Team         bool
	Organization string
	Project      string
}

// String is the redacted form; the credential is never part of it.
func (c Candidate) String() string {
	return "glm credential " + c.Plan + "/" + c.Region.Host()
}

// Key names of the store, from the ZCode CLI oauthCredentialRepo.
const (
	codingPlanPrefix  = "account-provider:coding-plan:account:"
	codingPlanSuffix  = ":api-key"
	codingPlanAccount = ":account:"
)

// Store is the parsed credential store: the file that was read and every
// credential the endpoint may accept, in the order the ZCode CLI would try
// them.
type Store struct {
	Path       string
	Candidates []Candidate
}

// Usable reports whether the store has a credential to try.
func (s Store) Usable() bool { return len(s.Candidates) > 0 }

// Load reads the credential store of home. A missing file is not an error
// while an environment key is set; with neither, it is ErrNoCredentials.
func Load(home string) (Store, error) {
	store := Store{Path: CredentialPath(home)}
	var stored []Candidate
	if store.Path != "" {
		data, err := os.ReadFile(store.Path)
		switch {
		case err == nil:
			values, decodeErr := decodeValues(data)
			if decodeErr != nil {
				return store, ErrCredentialFormat
			}
			stored = candidatesFromStore(values)
		case errors.Is(err, fs.ErrNotExist):
		default:
			return store, ErrCredentialFormat
		}
	}
	store.Candidates = append(envCandidates(), stored...)
	sortCandidates(store.Candidates)
	if !store.Usable() {
		return store, ErrNoCredentials
	}
	return store, nil
}

// decodeValues reads the flat string map of the store. A value that is not a
// string is skipped instead of failing the whole file.
func decodeValues(data []byte) (map[string]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	values := make(map[string]string, len(raw))
	for key, value := range raw {
		var text string
		if json.Unmarshal(value, &text) != nil {
			continue
		}
		values[key] = text
	}
	return values, nil
}

// candidatesFromStore collects the coding plan keys, then the OAuth access
// tokens. The names are visited in sorted order because a JSON map has none.
func candidatesFromStore(values map[string]string) []Candidate {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]Candidate, 0, len(names))
	for _, name := range names {
		plan, org, ok := codingPlanKey(name)
		if !ok {
			continue
		}
		key := strings.TrimSpace(values[name])
		if key == "" {
			continue
		}
		out = append(out, Candidate{
			Key: key, Kind: KindCodingPlan, Plan: plan, Region: regionOf(plan),
			Team: strings.Contains(plan, "team"), Organization: org,
		})
	}
	for _, provider := range []string{"bigmodel", "zai"} {
		key := strings.TrimSpace(values["oauth:"+provider+":access_token"])
		if key == "" {
			continue
		}
		out = append(out, Candidate{Key: key, Kind: KindOAuth, Plan: provider, Region: regionOf(provider)})
	}
	return out
}

// codingPlanKey splits
// account-provider:coding-plan:account:<plan>:account:<org>:api-key.
func codingPlanKey(name string) (plan, org string, ok bool) {
	if !strings.HasPrefix(name, codingPlanPrefix) || !strings.HasSuffix(name, codingPlanSuffix) {
		return "", "", false
	}
	middle := strings.TrimSuffix(strings.TrimPrefix(name, codingPlanPrefix), codingPlanSuffix)
	head, tail, _ := strings.Cut(middle, codingPlanAccount)
	head = strings.TrimSpace(head)
	if head == "" {
		return "", "", false
	}
	return head, strings.TrimSpace(tail), true
}

// regionOf maps a plan or provider id to its vendor site.
func regionOf(name string) Region {
	if strings.HasPrefix(name, "zai") {
		return Zai
	}
	return BigModel
}

// envKeys are the environment overrides of the ZCode CLI, in its order.
var envKeys = []string{"ZCODE_BIGMODEL_USAGE_API_KEY", "BIGMODEL_USAGE_API_KEY"}

// envCandidates returns the first environment key that is set.
func envCandidates() []Candidate {
	for _, name := range envKeys {
		key := strings.TrimSpace(os.Getenv(name))
		if key == "" {
			continue
		}
		return []Candidate{{Key: key, Kind: KindEnvKey, Plan: name, Region: BigModel}}
	}
	return nil
}

// sortCandidates fixes the try order: the domestic site first, then the
// environment key, then the stored coding plan keys, individual before team,
// and the OAuth tokens last.
func sortCandidates(list []Candidate) {
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		switch {
		case a.Region != b.Region:
			return a.Region < b.Region
		case a.Kind != b.Kind:
			return a.Kind < b.Kind
		case a.Team != b.Team:
			return !a.Team
		}
		return a.Plan < b.Plan
	})
}
