package mediaservers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	domain "github.com/HolyArrs/holyarrs/app/backend/internal/core/model"
	"github.com/HolyArrs/holyarrs/app/backend/internal/mediafiles"
)

// jellyLabelItem is one Movie or Series on a Jellyfin or Emby server matched
// to a Library title, together with its current tags.
type jellyLabelItem struct {
	id   string
	tags []string
}

// jellyAdminUser returns an administrator user id on a Jellyfin/Emby server
// (falling back to the first user), needed to read full editable items before
// posting updated tags.
func (l *Linker) jellyAdminUser(ctx context.Context, m domain.MediaServer) (string, error) {
	b, err := l.do(ctx, m, http.MethodGet, "/Users", nil)
	if err != nil {
		return "", err
	}
	var users []struct {
		ID     string `json:"Id"`
		Policy struct {
			IsAdministrator bool `json:"IsAdministrator"`
		} `json:"Policy"`
	}
	if err := json.Unmarshal(b, &users); err != nil {
		return "", errors.New("the server's answer was not a user list")
	}
	first := ""
	for _, u := range users {
		if u.ID == "" {
			continue
		}
		if first == "" {
			first = u.ID
		}
		if u.Policy.IsAdministrator {
			return u.ID, nil
		}
	}
	if first == "" {
		return "", errors.New("the server returned no users")
	}
	return first, nil
}

// jellyLabelItems walks every Movie and Series on a Jellyfin/Emby server and
// matches it against keys (kind|provider:value -> titleID).
func (l *Linker) jellyLabelItems(ctx context.Context, m domain.MediaServer, keys map[string]int64, each func(it jellyLabelItem, title int64)) error {
	for start := 0; ; start += watchPage {
		path := "/Items?Recursive=true&IncludeItemTypes=Movie,Series&Fields=ProviderIds,Tags&EnableImages=false&EnableUserData=false" +
			"&StartIndex=" + strconv.Itoa(start) + "&Limit=" + strconv.Itoa(watchPage)
		b, err := l.do(ctx, m, http.MethodGet, path, nil)
		if err != nil {
			return err
		}
		var r struct {
			Items []struct {
				ID          string            `json:"Id"`
				Type        string            `json:"Type"`
				ProviderIDs map[string]string `json:"ProviderIds"`
				Tags        []string
			}
		}
		if err := json.Unmarshal(b, &r); err != nil {
			return errors.New("the server's answer was not an item list")
		}
		for _, it := range r.Items {
			kind := "movie"
			if strings.EqualFold(it.Type, "Series") {
				kind = "show"
			}
			for k, v := range it.ProviderIDs {
				if k = strings.ToLower(k); v != "" && (k == "tvdb" || k == "tmdb" || k == "imdb") {
					if titleID := keys[kind+"|"+k+":"+v]; titleID != 0 {
						each(jellyLabelItem{id: it.ID, tags: it.Tags}, titleID)
						break
					}
				}
			}
		}
		if len(r.Items) < watchPage {
			return nil
		}
	}
}

// putJellyTags loads the full editable item JSON for itemID, appends add to
// its Tags array without losing existing metadata, and posts it back.
func (l *Linker) putJellyTags(ctx context.Context, m domain.MediaServer, userID, itemID string, add []string) error {
	b, err := l.do(ctx, m, http.MethodGet, "/Users/"+url.PathEscape(userID)+"/Items/"+url.PathEscape(itemID), nil)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return errors.New("the server's answer was not an item")
	}
	var tags []string
	seen := map[string]bool{}
	if raw, ok := doc["Tags"].([]any); ok {
		for _, x := range raw {
			if s, ok := x.(string); ok && s != "" {
				tags = append(tags, s)
				seen[strings.ToLower(s)] = true
			}
		}
	}
	changed := false
	for _, lb := range add {
		if !seen[strings.ToLower(lb)] {
			tags = append(tags, lb)
			seen[strings.ToLower(lb)] = true
			changed = true
		}
	}
	if !changed {
		return nil
	}
	doc["Tags"] = tags
	payload, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = l.do(ctx, m, http.MethodPost, "/Items/"+url.PathEscape(itemID), payload)
	return err
}

// labelJellyServer tags the asked-for titles on one Jellyfin or Emby server.
func (l *Linker) labelJellyServer(ctx context.Context, m domain.MediaServer, set LabelSettings, asked []AskedTitle) (int, error) {
	done, err := labelsDone(ctx, l.Store.Read(), m.ID)
	if err != nil {
		return 0, err
	}
	want := map[int64][]string{}
	for _, a := range asked {
		for _, lb := range labelsFor(set, a) {
			if !done[a.TitleID][strings.ToLower(lb)] {
				want[a.TitleID] = append(want[a.TitleID], lb)
			}
		}
	}
	if len(want) == 0 {
		return 0, nil
	}
	keys, err := mediafiles.TitlesBySourceID(ctx, l.Store.Read())
	if err != nil {
		return 0, err
	}
	items := map[int64][]jellyLabelItem{}
	if err := l.jellyLabelItems(ctx, m, keys, func(it jellyLabelItem, title int64) {
		if _, ok := want[title]; ok {
			items[title] = append(items[title], it)
		}
	}); err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}
	userID, err := l.jellyAdminUser(ctx, m)
	if err != nil {
		return 0, err
	}
	ids := make([]int64, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	n := 0
	for _, id := range ids {
		put := false
		for _, it := range items[id] {
			on := map[string]bool{}
			for _, h := range it.tags {
				on[strings.ToLower(h)] = true
			}
			var add []string
			for _, lb := range want[id] {
				if !on[strings.ToLower(lb)] {
					add = append(add, lb)
				}
			}
			if len(add) > 0 {
				if err := l.putJellyTags(ctx, m, userID, it.id, add); err != nil {
					return n, err
				}
				put = true
			}
		}
		if put {
			n++
		}
		if err := markLabels(ctx, l.Store, m.ID, id, want[id], time.Now().UTC()); err != nil {
			return n, err
		}
	}
	return n, nil
}
