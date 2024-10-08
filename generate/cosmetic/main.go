package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"regexp"
	"strings"
	"text/template"
	"time"

	"cosmetic/filter"
	"cosmetic/topdomains"
	"cosmetic/util"
)

func joinSorted(f []string, comma string) string {
	sort.Strings(f)
	return strings.Join(f, comma)
}

func joinSortedMeta(f []string, comma string) string {
	tmp := make([]string, len(f))
	copy(tmp, f)
	sort.Strings(tmp)
	for k, v := range tmp {
		tmp[k] = regexp.QuoteMeta(v)
	}
	return strings.Join(tmp, comma)
}

func toJSObject(x interface{}) string {
	b, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func main() {
	var (
		inputLists     = flag.String("input", "filter-lists.txt", "Path to file that defines URLs to blocklists")
		scriptTarget   = flag.String("output", "cosmetic.user.js", "Path to output file")
		topDomainsPath = flag.String("top", "", "Path to file downloaded from http://s3-us-west-1.amazonaws.com/umbrella-static/index.html")
		topDomainCount = flag.Int("topCount", 1_000_000, "Include up to this rank of highest-ranking top domains, only makes sense with -top")
	)
	flag.Parse()

	var topDomains *topdomains.TopDomainStorage
	if strings.TrimSpace(*topDomainsPath) != "" {
		tdm, err := topdomains.FromFile(*topDomainsPath, *topDomainCount)
		if err != nil {
			log.Fatalf("Reading top domains from file: %v", err)
		}
		topDomains = &tdm

		fmt.Printf("Read %d top domains\n", tdm.Len())
	}

	scriptTemplateContent, err := ioutil.ReadFile("script-template.js")
	if err != nil {
		log.Fatalf("reading script template file: %s\n", err.Error())
	}
	var scriptTemplate = template.Must(template.New("").Parse(string(scriptTemplateContent)))

	filterURLs, err := util.ReadListFile(*inputLists)
	if err != nil {
		log.Fatalf("cannot load list of filter URLs: %s\n", err.Error())
	}

	tempDir, err := ioutil.TempDir("", "cosmetic-filter-*")
	if err != nil {
		log.Fatalf("creating temp dir for cosmetic filters: %s\n", err.Error())
	}
	defer os.RemoveAll(tempDir)

	filterOutputFiles, err := util.DownloadURLs(filterURLs, tempDir)
	if err != nil {
		log.Fatalf("error downloading filter lists: %s\n", err.Error())
	}
	log.Printf("Downloaded %d filter files\n", len(filterOutputFiles))

	var filters []filter.Rule
	for _, fp := range filterOutputFiles {
		ff := util.FiltersFromFile(fp)
		if len(ff) == 0 {
			log.Printf("[Warning] No rules found in file %q\n", fp)
		}
		filters = append(filters, ff...)
	}
	fmt.Printf("Found %d filters in these files\n", len(filters))

	lookupTable := filter.Combine(filters)

	// This happens only for the "lite" version of the script
	if topDomains != nil {
		// Now only keep the filters for top/important domains
		topDomainLookupTable := make(map[string]filter.CombineResult)
		for domain, filter := range lookupTable {
			allTilded := true

			for _, fragment := range filter.Domains {
				if !strings.HasPrefix(fragment, "~") {
					allTilded = false

					if topDomains.Contains(fragment) {
						topDomainLookupTable[domain] = filter
						break
					}
				}
			}

			if allTilded {
				// I coded it but I decided to not include
				// topDomainLookupTable[domain] = filter
			}
		}
		fmt.Printf("Selected %d top domains from %d domains with available filters\n", len(topDomainLookupTable), len(lookupTable))

		// Also keep the default/general rules for all pages
		topDomainLookupTable[""] = lookupTable[""]

		lookupTable = topDomainLookupTable
	}

	var duplicateCount = map[string]int{}
	for _, f := range lookupTable {
		joined := joinSorted(f.Selectors, ",")
		duplicateCount[joined] = duplicateCount[joined] + 1
		joined = joinSorted(f.Exceptions, ",")
		duplicateCount[joined] = duplicateCount[joined] + 1
		joined = joinSorted(f.InjectedCSS, "")
		duplicateCount[joined] = duplicateCount[joined] + 1
		joined = joinSortedMeta(f.InjectionException, "|")
		duplicateCount[joined] = duplicateCount[joined] + 1
	}

	var deduplicatedStrings []string
	for f, count := range duplicateCount {
		if count > 1 {
			deduplicatedStrings = append(deduplicatedStrings, f)
		}
	}
	sort.Strings(deduplicatedStrings)

	var deduplicatedIndexMapping = map[string]int{}
	for i, r := range deduplicatedStrings {
		deduplicatedIndexMapping[r] = i
	}

	// The compiled rules are either
	// - a string, which is a css selector (usually selecting many elements)
	// - an int, which is the index of a common rule (that was present more than once)
	var (
		compiledSelectorRules  = map[string]interface{}{}
		compiledSelectorExceptions  = map[string]interface{}{}
		compiledInjectionRules = map[string]interface{}{}
		compiledInjectionExceptions = map[string]interface{}{}
	)
	for domain, filter := range lookupTable {
		if len(filter.Selectors) > 0 {
			joined := joinSorted(filter.Selectors, ",")
			if duplicateCount[joined] > 1 {
				compiledSelectorRules[domain] = deduplicatedIndexMapping[joined]
			} else {
				compiledSelectorRules[domain] = joined
			}
		}

		if len(filter.Exceptions) > 0 {
			joined := joinSorted(filter.Exceptions, ",")
			if duplicateCount[joined] > 1 {
				compiledSelectorExceptions[domain] = deduplicatedIndexMapping[joined]
			} else {
				compiledSelectorExceptions[domain] = joined
			}
		}

		if len(filter.InjectedCSS) > 0 {
			joined := joinSorted(filter.InjectedCSS, "")
			if duplicateCount[joined] > 1 {
				compiledInjectionRules[domain] = deduplicatedIndexMapping[joined]
			} else {
				compiledInjectionRules[domain] = joined
			}
		}

		if len(filter.InjectionException) > 0 {
			joined := joinSortedMeta(filter.InjectionException, "|")
			if duplicateCount[joined] > 1 {
				compiledInjectionExceptions[domain] = deduplicatedIndexMapping[joined]
			} else {
				compiledInjectionExceptions[domain] = joined
			}
		}
	}

	fmt.Printf("Combined them for %d domains\n", len(compiledSelectorRules))
	fmt.Printf("Exeption for %d domains\n", len(compiledSelectorExceptions))

	outputFile, err := os.Create(*scriptTarget)
	if err != nil {
		log.Fatalf("creating output file: %s\n", err.Error())
	}

	_, err = outputFile.WriteString("// THIS FILE IS AUTO-GENERATED. DO NOT EDIT. See generate/cosmetic directory for more info\n")
	if err != nil {
		log.Fatalf("could not write auto generated message: %s\n", err.Error())
	}

	err = scriptTemplate.Execute(outputFile, map[string]interface{}{
		"version":             time.Now().Format("2006.01.02"),
		"rules":               toJSObject(compiledSelectorRules),
		"exceptions":          toJSObject(compiledSelectorExceptions),
		"injectionRules":      toJSObject(compiledInjectionRules),
		"injectionExceptions": toJSObject(compiledInjectionExceptions),
		"deduplicatedStrings": toJSObject(deduplicatedStrings),
		"statistics":          fmt.Sprintf("blockers for %d domains, exceptions for %d domains, injected CSS rules for %d domains, exception for CSS injection for %d domains", len(compiledSelectorRules), len(compiledSelectorExceptions), len(compiledInjectionRules), len(compiledInjectionExceptions)),
		"isLite":              topDomains != nil,
		"topDomainCount":      *topDomainCount,
	})
	if err != nil {
		log.Fatalf("Error generating script text: %s\n", err.Error())
	}

	err = outputFile.Close()
	if err != nil {
		log.Fatalf("could not close output file: %s\n", err.Error())
	}
}
