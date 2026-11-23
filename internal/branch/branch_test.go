package branch

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"testing"
)

func TestPlanNewStoryFromDev(t *testing.T) {
	got, err := Derive("Add saved cards\n\nAs a shopper I save a card.", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "feat/add-saved-cards" || got.Base != "dev" || got.Kind != "feat" {
		t.Fatalf("%+v", got)
	}
}

func TestPlanLinkedStoryKeepsKey(t *testing.T) {
	got, err := Derive("Show the channel list", "AET-1478", "feat", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "feat/AET-1478-show-the-channel-list" || got.Base != "dev" {
		t.Fatalf("%+v", got)
	}
}

func TestPlanKeyInsideTheStory(t *testing.T) {
	got, err := Derive("AET-1478 Show channels", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "feat/AET-1478-show-channels" {
		t.Fatalf("%+v", got)
	}
}

func TestPlanFixFromDevAndHotfixFromMaster(t *testing.T) {
	fix, err := Derive("fix: ledger rounds the wrong way", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if fix.Name != "fix/ledger-rounds-the-wrong-way" || fix.Base != "dev" {
		t.Fatalf("%+v", fix)
	}
	hot, err := Derive("refund the double charge", "", "", "master")
	if err != nil {
		t.Fatal(err)
	}
	if hot.Name != "hotfix/refund-the-double-charge" || hot.Base != "master" || hot.Kind != "hotfix" {
		t.Fatalf("%+v", hot)
	}
}

func TestPlanRejectsOtherPrefixes(t *testing.T) {
	if _, err := Derive("Add saved cards", "", "chore", ""); err == nil {
		t.Fatal("chore was accepted")
	}
}

// An explicit base is the person's choice: stacked work starts from a feature branch, and
// a feat may start from master. Only the default still depends on the kind.
func TestDeriveBase(t *testing.T) {
	cases := []struct {
		name, story, kind, base string
		wantName, wantBase      string
		wantErr                 bool
	}{
		{name: "feat default", story: "Add saved cards", wantName: "feat/add-saved-cards", wantBase: "dev"},
		{name: "fix default", story: "fix: rounding", wantName: "fix/rounding", wantBase: "dev"},
		{name: "hotfix default", story: "refund", kind: "hotfix", wantName: "hotfix/refund", wantBase: "master"},
		{name: "master infers hotfix", story: "refund", base: "master", wantName: "hotfix/refund", wantBase: "master"},
		{name: "stacked on a feature", story: "Add saved cards", base: "feat/SAL-3-wallet", wantName: "feat/add-saved-cards", wantBase: "feat/SAL-3-wallet"},
		{name: "feat from master", story: "Add saved cards", kind: "feat", base: "master", wantName: "feat/add-saved-cards", wantBase: "master"},
		{name: "hotfix from release", story: "refund", kind: "hotfix", base: "release/2.1", wantName: "hotfix/refund", wantBase: "release/2.1"},
		{name: "space", story: "x", base: "my base", wantErr: true},
		{name: "dot dot", story: "x", base: "a..b", wantErr: true},
		{name: "leading dash", story: "x", base: "-dev", wantErr: true},
		{name: "leading slash", story: "x", base: "/dev", wantErr: true},
		{name: "trailing slash", story: "x", base: "dev/", wantErr: true},
		{name: "lock suffix", story: "x", base: "dev.lock", wantErr: true},
		{name: "reflog", story: "x", base: "dev@{1}", wantErr: true},
		{name: "tilde", story: "x", base: "dev~1", wantErr: true},
		{name: "caret", story: "x", base: "dev^", wantErr: true},
		{name: "colon", story: "x", base: "a:b", wantErr: true},
		{name: "glob", story: "x", base: "de*v", wantErr: true},
		{name: "bracket", story: "x", base: "de[v", wantErr: true},
		{name: "backslash", story: "x", base: `de\v`, wantErr: true},
		{name: "question", story: "x", base: "de?v", wantErr: true},
		{name: "itself", story: "Add saved cards", base: "feat/add-saved-cards", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Derive(tc.story, "", tc.kind, tc.base)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("base %q was accepted: %+v", tc.base, got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != tc.wantName || got.Base != tc.wantBase {
				t.Fatalf("%+v", got)
			}
			if !Valid(got.Name, got.Base) {
				t.Fatalf("Valid(%q, %q) is false", got.Name, got.Base)
			}
		})
	}
}

func TestValidRefFollowsCheckRefFormat(t *testing.T) {
	for _, ok := range []string{"dev", "master", "feat/SAL-3", "release/2.1", "a.b"} {
		if !ValidRef(ok) {
			t.Errorf("%q was rejected", ok)
		}
	}
	for _, bad := range []string{"", "@", "HEAD", "a b", "a..b", "-a", "/a", "a/", "a.lock", "a@{b", "a~", "a^", "a:", "a?", "a*", "a[", `a\b`, "a//b", ".a", "a/.b", "a.", "a\x7f"} {
		if ValidRef(bad) {
			t.Errorf("%q was accepted", bad)
		}
	}
}

func TestASpanishStoryStillProducesAValidBranch(t *testing.T) {
	story := "SAL-4 Retirar de Hermes los módulos que no se usan y dejar la navegación limpia\nType: Story\n"
	plan, err := Derive(story, "", "", "")
	if err != nil {
		t.Fatalf("a Spanish title was rejected: %v", err)
	}
	if !Valid(plan.Name, plan.Base) {
		t.Fatalf("derived name is not valid: %q", plan.Name)
	}
	for _, r := range plan.Name {
		if r > 127 {
			t.Fatalf("branch %q kept a non-ASCII letter", plan.Name)
		}
	}
	if strings.Contains(plan.Name, "modulos") == false {
		t.Fatalf("the accent was dropped instead of folded: %q", plan.Name)
	}
}

func TestTheBranchBudgetIsTheTitleNotTheWholeSlug(t *testing.T) {
	story := "SAL-4 Retirar de Hermes los módulos que no se usan y dejar la navegación limpia\nType: Story\n"
	plan, err := Derive(story, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	title := strings.TrimPrefix(plan.Name, "feat/SAL-4-")
	if len([]rune(title)) > 48 {
		t.Fatalf("title is %d runes: %q", len([]rune(title)), title)
	}
	if strings.HasSuffix(title, "-") {
		t.Fatalf("the title ends with a dash: %q", title)
	}
	if plan.Feature != strings.ToLower("SAL-4-"+title) {
		t.Fatalf("feature = %q", plan.Feature)
	}
}

func TestTheFeatureNamePassesTheSameRuleAsThePolicy(t *testing.T) {
	plan, err := Derive("SAL-4 Módulos y navegación", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range plan.Feature {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
		default:
			t.Fatalf("feature %q has %q", plan.Feature, r)
		}
	}
}

// The exact story that failed: the accent in "módulos" made the derived name invalid.
func TestTheStoryThatFailedNowDerives(t *testing.T) {
	story := "SAL-4 Retirar de Hermes los módulos que no se usan\nType: Story\n"
	plan, err := Derive(story, "", "", "")
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if plan.Name != "feat/SAL-4-retirar-de-hermes-los-modulos-que-no-se-usan" {
		t.Fatalf("name = %q", plan.Name)
	}
	if plan.Feature != "sal-4-retirar-de-hermes-los-modulos-que-no-se-usan" {
		t.Fatalf("feature = %q", plan.Feature)
	}
}

func TestValidBaseRefusesOptionsHeadAndCommits(t *testing.T) {
	for _, ok := range []string{"dev", "master", "feat/SAL-3", "release/2.1", "x", "deadbee-x"} {
		if err := CheckBase(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "--upload-pack=touch /tmp/PWNED1", "--repo=evil/x", "-x", "HEAD", "@", "deadbee", "0123456789abcdef0123456789abcdef01234567", " dev", "dev ", "a..b"} {
		err := CheckBase(bad)
		if err == nil || !strings.Contains(err.Error(), "invalid base branch") {
			t.Errorf("%q: err = %v", bad, err)
		}
		if _, err := Derive("Add saved cards", "", "", bad); bad != "" && err == nil {
			t.Errorf("Derive accepted base %q", bad)
		}
	}
}

// A project base: line is a default for feat and fix only. It never makes a story a hotfix,
// and a hotfix keeps master. A base the run names is the person's choice for any kind; only
// an explicit master without a kind still means hotfix, as it always has.
func TestChooseKindAndBase(t *testing.T) {
	cases := []struct {
		name, story, kind, explicit, project string
		wantKind, wantBase, wantName         string
	}{
		{name: "nothing chosen", story: "Add cards", wantKind: "", wantBase: "", wantName: "feat/add-cards"},
		{name: "project base for feat", story: "Add cards", project: "release/2.1", wantKind: "feat", wantBase: "release/2.1", wantName: "feat/add-cards"},
		{name: "project base for fix", story: "fix: rounding", project: "release/2.1", wantKind: "fix", wantBase: "release/2.1", wantName: "fix/rounding"},
		{name: "project dev leaves hotfix on master", story: "refund", kind: "hotfix", project: "dev", wantKind: "hotfix", wantBase: "", wantName: "hotfix/refund"},
		{name: "project master never infers hotfix", story: "Add cards", project: "master", wantKind: "feat", wantBase: "master", wantName: "feat/add-cards"},
		{name: "hotfix story keeps master", story: "hotfix: refund", project: "dev", wantKind: "", wantBase: "", wantName: "hotfix/refund"},
		{name: "explicit wins over project", story: "Add cards", explicit: "feat/SAL-3", project: "dev", wantKind: "", wantBase: "feat/SAL-3", wantName: "feat/add-cards"},
		{name: "explicit for hotfix", story: "refund", kind: "hotfix", explicit: "release/2.1", wantKind: "hotfix", wantBase: "release/2.1", wantName: "hotfix/refund"},
		{name: "explicit master still infers hotfix", story: "refund", explicit: "master", wantKind: "", wantBase: "master", wantName: "hotfix/refund"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, base, err := Choose(tc.story, tc.kind, tc.explicit, tc.project)
			if err != nil {
				t.Fatal(err)
			}
			if kind != tc.wantKind || base != tc.wantBase {
				t.Fatalf("kind %q base %q", kind, base)
			}
			plan, err := Derive(tc.story, "", kind, base)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Name != tc.wantName {
				t.Fatalf("name %q", plan.Name)
			}
		})
	}
	if _, _, err := Choose("x", "", "--repo=evil/x", ""); err == nil {
		t.Fatal("an option was accepted as an explicit base")
	}
	if _, _, err := Choose("x", "", "", "HEAD"); err == nil {
		t.Fatal("HEAD was accepted as a project base")
	}
}

func shortHash(name string) string {
	sum := sha1.Sum([]byte(name))
	return hex.EncodeToString(sum[:])[:6]
}

func TestFeatureOfNamesEveryBranchApart(t *testing.T) {
	long := "feat/" + strings.Repeat("a", 120)
	for name, want := range map[string]string{
		"feat/a.b":              "a.b",
		"fix/a.b":               "fix-a.b",
		"hotfix/a.b":            "hotfix-a.b",
		"feat/sal-1-x":          "sal-1-x",
		"feat/SAL-1-x":          "sal-1-x",
		"feat/x-y":              "x-y",
		"feat/x/y":              "x-y-" + shortHash("feat/x/y"),
		"feat/fix-a.b":          "fix-a.b-" + shortHash("feat/fix-a.b"),
		"fix/Añadir tarjeta":    "fix-anadir-tarjeta-" + shortHash("fix/Añadir tarjeta"),
		"  feat/sal-9-refund  ": "sal-9-refund",
		"feat/con":              "con-",
		"feat/nul.x":            "nul-.x",
		"feat/com1":             "com1-",
		long:                    strings.Repeat("a", 73) + "-" + shortHash(long),
		"dev":                   "",
		"main":                  "",
		"HEAD":                  "",
		"chore/x":               "",
		"feat/":                 "",
		"feat/..":               "",
		"":                      "",
	} {
		if got := FeatureOf(name); got != want {
			t.Fatalf("FeatureOf(%q) = %q, want %q", name, got, want)
		}
	}
	pairs := [][2]string{
		{"feat/a.b", "fix/a.b"}, {"feat/a.b", "hotfix/a.b"}, {"fix/a.b", "hotfix/a.b"},
		{"feat/x/y", "feat/x-y"}, {"feat/fix-a.b", "fix/a.b"},
	}
	for _, pair := range pairs {
		if FeatureOf(pair[0]) == FeatureOf(pair[1]) {
			t.Fatalf("%s and %s share the feature %q", pair[0], pair[1], FeatureOf(pair[0]))
		}
	}
	if got := FeatureOf(long); len(got) != 80 || !ValidFeature(got) {
		t.Fatalf("long feature %q (%d)", got, len(got))
	}
	if !ValidFeature("sal-4.cards_x") || ValidFeature("../x") || ValidFeature("") || ValidFeature("..") {
		t.Fatal("ValidFeature does not follow the feature rule")
	}
}

// Span numbers acceptance criteria AC-1, AC-2. Those match the shape of a Jira key, so a
// story with criteria must not be filed under one of them.
func TestAcceptanceCriterionIdsAreNotStoryKeys(t *testing.T) {
	story := "Add a Median function.\nAcceptance criteria:\n1. AC-1 Median of 1,3,2 is 2."
	plan, err := Derive(story, "", Feat, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plan.Name, "AC-1") {
		t.Fatalf("a criterion id became the story key: %s", plan.Name)
	}
	keyed, err := Derive("SAL-3 Add a Median function. AC-1 it works", "", Feat, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(keyed.Name, "SAL-3") {
		t.Fatalf("a real key was lost: %s", keyed.Name)
	}
}
