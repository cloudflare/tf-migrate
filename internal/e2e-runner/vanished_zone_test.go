package e2e

import (
	"reflect"
	"testing"
)

// realPlanFailure is trimmed verbatim from the Supportability Matrix run that
// surfaced this failure mode (GitHub Actions run 41699862833), where 21 of 23
// cloudflare_zone entries in the shared v4 base referred to zones that had
// been deleted out of band. Keeping the real shape — blank line between the
// Error: line and the `with` line, interleaved warnings, indexed addresses —
// is the point of this fixture.
const realPlanFailure = `
module.access_rule.cloudflare_access_rule.target_country: Refreshing state... [id=a8db18bf45eb4e3b8d3b1d01a9ae7415]
module.zone.cloudflare_zone.minimal: Refreshing state... [id=3176fd9bc2858ecabf3caafa4d7a6734]

Warning: Argument is deprecated

  with module.argo.cloudflare_argo.both_with_lifecycle,
  on argo/argo.tf line 3, in resource "cloudflare_argo" "both_with_lifecycle":
   3: resource "cloudflare_argo" "both_with_lifecycle" {

Error: error finding Zone "3176fd9bc2858ecabf3caafa4d7a6734": Invalid zone identifier (1001)

  with module.zone.cloudflare_zone.minimal,
  on zone/zone.tf line 10, in resource "cloudflare_zone" "minimal":
   10: resource "cloudflare_zone" "minimal" {


Error: error finding Zone "51f6af7041175be288aff893a6b0f177": Invalid zone identifier (1001)

  with module.zone.cloudflare_zone.counted_zones[1],
  on zone/zone.tf line 101, in resource "cloudflare_zone" "counted_zones":
  101: resource "cloudflare_zone" "counted_zones" {


Error: error finding Zone "ee18ad7e5f81830dd00db7e80a4d30c4": Invalid zone identifier (1001)

  with module.zone.cloudflare_zone.set_zones["cftftest5190-delta.cf-tf-test.com"],
  on zone/zone.tf line 84, in resource "cloudflare_zone" "set_zones":
   84: resource "cloudflare_zone" "set_zones" {
`

func TestParseVanishedZones_RealPlanFailure(t *testing.T) {
	got := parseVanishedZones(realPlanFailure)
	want := []vanishedZone{
		{Address: "module.zone.cloudflare_zone.minimal", ZoneID: "3176fd9bc2858ecabf3caafa4d7a6734"},
		{Address: "module.zone.cloudflare_zone.counted_zones[1]", ZoneID: "51f6af7041175be288aff893a6b0f177"},
		{Address: `module.zone.cloudflare_zone.set_zones["cftftest5190-delta.cf-tf-test.com"]`, ZoneID: "ee18ad7e5f81830dd00db7e80a4d30c4"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseVanishedZones()\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParseVanishedZones(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "no errors at all",
			in:   "module.zone.cloudflare_zone.minimal: Refreshing state...\n\nNo changes. Your infrastructure matches the configuration.\n",
			want: nil,
		},
		{
			name: "unrelated error is not matched",
			in: `
Error: failed to create DNS record: An A, AAAA, CNAME record with that host already exists.

  with module.dns_record.cloudflare_record.basic,
`,
			want: nil,
		},
		{
			name: "zone-not-found reported against a non-zone resource is left alone",
			in: `
Error: error finding Zone "deadbeefdeadbeefdeadbeefdeadbeef": Invalid zone identifier (1001)

  with module.dns_record.cloudflare_record.in_deleted_zone,
`,
			want: nil,
		},
		{
			name: "address without a module prefix",
			in: `
Error: error finding Zone "deadbeefdeadbeefdeadbeefdeadbeef": Invalid zone identifier (1001)

  with cloudflare_zone.minimal,
`,
			want: []string{"cloudflare_zone.minimal"},
		},
		{
			name: "duplicate addresses are collapsed",
			in: `
Error: error finding Zone "aaa": Invalid zone identifier (1001)

  with module.zone.cloudflare_zone.minimal,

Error: error finding Zone "aaa": Invalid zone identifier (1001)

  with module.zone.cloudflare_zone.minimal,
`,
			want: []string{"module.zone.cloudflare_zone.minimal"},
		},
		{
			name: "error with no attributed address is dropped, next one still parsed",
			in: `
Error: error finding Zone "aaa": Invalid zone identifier (1001)

Error: error finding Zone "bbb": Invalid zone identifier (1001)

  with module.zone.cloudflare_zone.paused_zone,
`,
			want: []string{"module.zone.cloudflare_zone.paused_zone"},
		},
		{
			name: "a warning between the error and its address stops attribution",
			in: `
Error: error finding Zone "aaa": Invalid zone identifier (1001)

Warning: Argument is deprecated

  with module.argo.cloudflare_argo.both_with_lifecycle,
`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, z := range parseVanishedZones(tt.in) {
				got = append(got, z.Address)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseVanishedZones() addresses\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestPluralizeEntry(t *testing.T) {
	if got := pluralizeEntry(1); got != "y" {
		t.Errorf("pluralizeEntry(1) = %q, want %q", got, "y")
	}
	for _, n := range []int{0, 2, 21} {
		if got := pluralizeEntry(n); got != "ies" {
			t.Errorf("pluralizeEntry(%d) = %q, want %q", n, got, "ies")
		}
	}
}

// realStateExcerpt mirrors the shape of `terraform state pull` output for the
// v4 base at the point the zones vanished: two cloudflare_record cross-refs
// that lived inside the deleted "minimal" zone, a record in the surviving
// main test zone, and an indexed resource. Terraform never reported the
// cross-refs in the failing plan — it skips refreshing a resource whose
// dependency already errored — so they only surfaced after the zone entries
// were removed.
const realStateExcerpt = `{
  "version": 4,
  "serial": 480,
  "resources": [
    {
      "module": "module.zone",
      "mode": "managed",
      "type": "cloudflare_zone",
      "name": "minimal",
      "instances": [{"attributes": {"id": "e97c42c902f4c122a82b18f26bdbb63e", "zone": "cftftest5190-minimal.cf-tf-test.com"}}]
    },
    {
      "module": "module.zone",
      "mode": "managed",
      "type": "cloudflare_record",
      "name": "cftftest5190_zone_xref",
      "instances": [{"attributes": {"id": "rec1", "zone_id": "e97c42c902f4c122a82b18f26bdbb63e"}}]
    },
    {
      "module": "module.zone",
      "mode": "managed",
      "type": "cloudflare_record",
      "name": "cftftest5190_zone_xref_interp",
      "instances": [{"attributes": {"id": "rec2", "zone_id": "e97c42c902f4c122a82b18f26bdbb63e"}}]
    },
    {
      "module": "module.dns_record",
      "mode": "managed",
      "type": "cloudflare_record",
      "name": "in_live_zone",
      "instances": [{"attributes": {"id": "rec3", "zone_id": "cd581854c1f59f8c686ee796d0eddce2"}}]
    },
    {
      "module": "module.zone",
      "mode": "managed",
      "type": "cloudflare_record",
      "name": "indexed",
      "instances": [
        {"index_key": "a", "attributes": {"id": "rec4", "zone_id": "e97c42c902f4c122a82b18f26bdbb63e"}},
        {"index_key": 2, "attributes": {"id": "rec5", "zone_id": "e97c42c902f4c122a82b18f26bdbb63e"}}
      ]
    },
    {
      "mode": "data",
      "type": "cloudflare_zone",
      "name": "by_id",
      "instances": [{"attributes": {"id": "cd581854c1f59f8c686ee796d0eddce2"}}]
    }
  ]
}`

func TestFindResourcesInZones(t *testing.T) {
	dead := map[string]bool{"e97c42c902f4c122a82b18f26bdbb63e": true}
	skip := map[string]bool{"module.zone.cloudflare_zone.minimal": true}

	got, err := findResourcesInZones(realStateExcerpt, dead, skip)
	if err != nil {
		t.Fatalf("findResourcesInZones() error = %v", err)
	}
	want := []string{
		"module.zone.cloudflare_record.cftftest5190_zone_xref",
		"module.zone.cloudflare_record.cftftest5190_zone_xref_interp",
		`module.zone.cloudflare_record.indexed["a"]`,
		"module.zone.cloudflare_record.indexed[2]",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findResourcesInZones()\n got: %#v\nwant: %#v", got, want)
	}
}

func TestFindResourcesInZones_NoDeadZones(t *testing.T) {
	got, err := findResourcesInZones(realStateExcerpt, map[string]bool{}, nil)
	if err != nil {
		t.Fatalf("findResourcesInZones() error = %v", err)
	}
	if got != nil {
		t.Errorf("expected no addresses for an empty dead-zone set, got %#v", got)
	}
}

func TestFindResourcesInZones_LiveZoneUntouched(t *testing.T) {
	// The surviving main test zone must never be swept up.
	got, err := findResourcesInZones(realStateExcerpt, map[string]bool{"e97c42c902f4c122a82b18f26bdbb63e": true}, nil)
	if err != nil {
		t.Fatalf("findResourcesInZones() error = %v", err)
	}
	for _, addr := range got {
		if addr == "module.dns_record.cloudflare_record.in_live_zone" {
			t.Fatalf("resource in a live zone was selected for removal: %s", addr)
		}
	}
}

func TestFindResourcesInZones_BadJSON(t *testing.T) {
	if _, err := findResourcesInZones("{not json", map[string]bool{"x": true}, nil); err == nil {
		t.Error("expected an error for unparsable state")
	}
}
