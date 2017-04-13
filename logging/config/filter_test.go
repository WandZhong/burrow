package config

import (
	"testing"

	. "github.com/hyperledger/burrow/util/slice"
	"github.com/stretchr/testify/assert"
)

func TestBuildKeyValuesPredicateMatchAll(t *testing.T) {
	conf := []*KeyValuePredicateConfig{
		{
			KeyRegex:   "Foo",
			ValueRegex: "bar",
		},
	}
	kvp, err := BuildKeyValuesPredicate(conf, true)
	assert.NoError(t, err)
	assert.True(t, kvp(Slice("Foo", "bar", "Bosh", "Bish")))
}

func TestBuildKeyValuesPredicateMatchAny(t *testing.T) {
	conf := []*KeyValuePredicateConfig{
		{
			KeyRegex:   "Bosh",
			ValueRegex: "Bish",
		},
	}
	kvp, err := BuildKeyValuesPredicate(conf, false)
	assert.NoError(t, err)
	assert.True(t, kvp(Slice("Foo", "bar", "Bosh", "Bish")))
}

func TestExcludeAllFilterPredicate(t *testing.T) {
	fc := &FilterConfig{
		FilterMode: ExcludeWhenAllMatch,
		Predicates: []*KeyValuePredicateConfig{
			{
				KeyRegex:   "Bosh",
				ValueRegex: "Bish",
			},
			{
				KeyRegex:   "Bosh",
				ValueRegex: "Bash",
			},
		},
	}
	fp, err := BuildFilterPredicate(fc)
	assert.NoError(t, err)
	assert.False(t, fp(Slice("Bosh", "Bash", "Shoes", 42)))
	assert.True(t, fp(Slice("Bosh", "Bash", "Foo", "bar", "Shoes", 42, "Bosh", "Bish")))
	assert.False(t, fp(Slice("Food", 0.2, "Shoes", 42)))

}
func TestExcludeAnyFilterPredicate(t *testing.T) {
	fc := &FilterConfig{
		FilterMode: ExcludeWhenAnyMatches,
		Predicates: []*KeyValuePredicateConfig{
			{
				KeyRegex:   "Bosh",
				ValueRegex: "Bish",
			},
			{
				KeyRegex:   "Bosh",
				ValueRegex: "Bash",
			},
		},
	}
	fp, err := BuildFilterPredicate(fc)
	assert.NoError(t, err)
	assert.False(t, fp(Slice("Foo", "bar", "Shoes", 42)))
	assert.True(t, fp(Slice("Foo", "bar", "Shoes", 42, "Bosh", "Bish")))
	assert.True(t, fp(Slice("Food", 0.2, "Shoes", 42, "Bosh", "Bish")))

}

func TestIncludeAllFilterPredicate(t *testing.T) {
	fc := &FilterConfig{
		FilterMode: IncludeWhenAllMatch,
		Predicates: []*KeyValuePredicateConfig{
			{
				KeyRegex:   "Bosh",
				ValueRegex: "Bish",
			},
			{
				KeyRegex:   "Planks",
				ValueRegex: "^0.2$",
			},
		},
	}
	fp, err := BuildFilterPredicate(fc)
	assert.NoError(t, err)
	assert.True(t, fp(Slice("Foo", "bar", "Shoes", 42)))
	// Don't filter, it has all the required key values
	assert.False(t, fp(Slice("Foo", "bar", "Planks", 0.2, "Shoes", 42, "Bosh", "Bish")))
	assert.True(t, fp(Slice("Food", 0.2, "Shoes", 42)))
}

func TestIncludeAnyFilterPredicate(t *testing.T) {
	fc := &FilterConfig{
		FilterMode: IncludeWhenAnyMatches,
		Predicates: []*KeyValuePredicateConfig{
			{
				KeyRegex:   "Bosh",
				ValueRegex: "Bish",
			},
			{
				KeyRegex:   "^Shoes$",
				ValueRegex: "42",
			},
		},
	}
	fp, err := BuildFilterPredicate(fc)
	assert.NoError(t, err)
	assert.False(t, fp(Slice("Foo", "bar", "Shoes", 3427)))
	assert.False(t, fp(Slice("Foo", "bar", "Shoes", 42, "Bosh", "Bish")))
	assert.False(t, fp(Slice("Food", 0.2, "Shoes", 42)))

}
