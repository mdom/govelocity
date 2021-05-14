package main

import (
    "strings"
)

type index struct {
    ngrams map[string]map[int]bool
    len    int
}

func newIndex() *index {
    return &index{ngrams: make(map[string]map[int]bool)}
}

func generate_trigrams(s string) []string {
    trigrams := []string{}
    s = strings.ToLower(s)
    words := strings.Fields(s)
    for _, word := range words {
        l := len(word)
        for i := 0; i <= l-3; i++ {
            trigrams = append(trigrams, word[i:i+3])
        }
    }
    return trigrams
}

func (i *index) add(content string) {
    trigrams := generate_trigrams(content)
    for _, tg := range trigrams {
        if _, ok := i.ngrams[tg]; !ok {
            i.ngrams[tg] = make(map[int]bool)
        }
        i.ngrams[tg][i.len] = true
    }
    i.len++
}

func (i *index) search(search string) []int {
    documentID := make(map[int]int)
    trigrams := generate_trigrams(search)
    for _, tg := range trigrams {
        if val, ok := i.ngrams[tg]; ok {
            for k := range val {
                documentID[k]++
            }
        }
    }
    intersection := []int{}
    for k, v := range documentID {
        if v == len(trigrams) {
            intersection = append(intersection, k)
        }
    }
    return intersection
}
