package main

import "time"

type DnsRecords struct {
	Result []DnsResult `json:"result"`
}
type DnsResult struct {
	Comment           string        `json:"comment"`
	Name              string        `json:"name"`
	Proxied           bool          `json:"proxied"`
	Id                string        `json:"id"`
	Tags              []interface{} `json:"tags"`
	Ttl               int           `json:"ttl"`
	Content           string        `json:"content"`
	Type              string        `json:"type"`
	CommentModifiedOn time.Time     `json:"comment_modified_on"`
	CreatedOn         time.Time     `json:"created_on"`
	ModifiedOn        time.Time     `json:"modified_on"`
	Proxiable         bool          `json:"proxiable"`
	TagsModifiedOn    time.Time     `json:"tags_modified_on"`
}

type DnsPayload struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	Ttl     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}
