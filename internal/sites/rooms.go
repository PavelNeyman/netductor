package sites

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

const MaxRoomPhotos = 5

var roomIDRe = regexp.MustCompile(`^[a-z0-9-]{1,32}$`)

// Room is a zone inside a Site (inventory + links to cameras / edge).
type Room struct {
	ID          string   `json:"id"`
	SiteID      string   `json:"site_id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	CameraIDs   []string `json:"camera_ids,omitempty"`
	EdgeIDs     []string `json:"edge_ids,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	PhotoCount  int      `json:"photo_count"` // 0..MaxRoomPhotos
	Updated     int64    `json:"updated"`
}

func roomsDir(siteID string) string {
	return filepath.Join(paths.StateDir(), "sites", siteID, "rooms")
}

func roomJSONPath(siteID, roomID string) string {
	return filepath.Join(roomsDir(siteID), roomID+".json")
}

func roomPhotoDir(siteID, roomID string) string {
	return filepath.Join(roomsDir(siteID), roomID)
}

func photoPath(siteID, roomID string, idx int) string {
	return filepath.Join(roomPhotoDir(siteID, roomID), fmt.Sprintf("%d.jpg", idx))
}

func ValidateRoomID(id string) error {
	if !roomIDRe.MatchString(id) {
		return fmt.Errorf("room id must match [a-z0-9-]{1,32}")
	}
	return nil
}

func ListRooms(siteID string) ([]Room, error) {
	if siteID == "" {
		return nil, fmt.Errorf("site_id required")
	}
	mu.Lock()
	defer mu.Unlock()
	dir := roomsDir(siteID)
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Room{}, nil
		}
		return nil, err
	}
	var out []Room
	for _, e := range ents {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var r Room
		if json.Unmarshal(b, &r) != nil {
			continue
		}
		r.PhotoCount = countPhotosLocked(siteID, r.ID)
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func GetRoom(siteID, roomID string) (Room, bool) {
	mu.Lock()
	defer mu.Unlock()
	return getRoomLocked(siteID, roomID)
}

func getRoomLocked(siteID, roomID string) (Room, bool) {
	b, err := os.ReadFile(roomJSONPath(siteID, roomID))
	if err != nil {
		return Room{}, false
	}
	var r Room
	if json.Unmarshal(b, &r) != nil {
		return Room{}, false
	}
	r.PhotoCount = countPhotosLocked(siteID, roomID)
	return r, true
}

func countPhotosLocked(siteID, roomID string) int {
	n := 0
	for i := 0; i < MaxRoomPhotos; i++ {
		if _, err := os.Stat(photoPath(siteID, roomID, i)); err == nil {
			n++
		}
	}
	return n
}

func UpsertRoom(r Room) (Room, error) {
	if r.SiteID == "" {
		return r, fmt.Errorf("site_id required")
	}
	if err := ValidateRoomID(r.ID); err != nil {
		return r, err
	}
	if r.Name == "" {
		r.Name = r.ID
	}
	// site must exist (Get takes its own lock)
	if _, ok := Get(r.SiteID); !ok {
		return r, fmt.Errorf("site %q not found — create site first", r.SiteID)
	}
	mu.Lock()
	defer mu.Unlock()
	_ = os.MkdirAll(roomsDir(r.SiteID), 0o700)
	r.Updated = time.Now().Unix()
	r.PhotoCount = countPhotosLocked(r.SiteID, r.ID)
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return r, err
	}
	if err := os.WriteFile(roomJSONPath(r.SiteID, r.ID), append(raw, '\n'), 0o600); err != nil {
		return r, err
	}
	return r, nil
}

func DeleteRoom(siteID, roomID string) error {
	if siteID == "" || roomID == "" {
		return fmt.Errorf("site_id and room id required")
	}
	mu.Lock()
	defer mu.Unlock()
	_ = os.Remove(roomJSONPath(siteID, roomID))
	_ = os.RemoveAll(roomPhotoDir(siteID, roomID))
	return nil
}

// AddPhoto appends a JPEG/PNG (converted stored as .jpg name; bytes as-is). Max MaxRoomPhotos.
func AddPhoto(siteID, roomID string, data []byte, contentType string) (int, error) {
	if len(data) == 0 {
		return -1, fmt.Errorf("empty photo")
	}
	if len(data) > 2<<20 {
		return -1, fmt.Errorf("photo too large (max 2 MiB)")
	}
	ct := strings.ToLower(contentType)
	if ct != "" && !strings.Contains(ct, "jpeg") && !strings.Contains(ct, "jpg") && !strings.Contains(ct, "png") {
		return -1, fmt.Errorf("only jpeg/png allowed")
	}
	// magic sniff
	if len(data) >= 3 && !(data[0] == 0xff && data[1] == 0xd8) && !(data[0] == 0x89 && data[1] == 'P') {
		return -1, fmt.Errorf("not jpeg/png payload")
	}
	mu.Lock()
	defer mu.Unlock()
	if _, ok := getRoomLocked(siteID, roomID); !ok {
		return -1, fmt.Errorf("room not found")
	}
	idx := -1
	for i := 0; i < MaxRoomPhotos; i++ {
		if _, err := os.Stat(photoPath(siteID, roomID, i)); os.IsNotExist(err) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return -1, fmt.Errorf("max %d photos per room", MaxRoomPhotos)
	}
	if err := os.MkdirAll(roomPhotoDir(siteID, roomID), 0o700); err != nil {
		return -1, err
	}
	if err := os.WriteFile(photoPath(siteID, roomID, idx), data, 0o600); err != nil {
		return -1, err
	}
	// bump updated on room json
	if r, ok := getRoomLocked(siteID, roomID); ok {
		r.PhotoCount = countPhotosLocked(siteID, roomID)
		r.Updated = time.Now().Unix()
		raw, _ := json.MarshalIndent(r, "", "  ")
		_ = os.WriteFile(roomJSONPath(siteID, roomID), append(raw, '\n'), 0o600)
	}
	return idx, nil
}

func GetPhoto(siteID, roomID string, idx int) ([]byte, error) {
	if idx < 0 || idx >= MaxRoomPhotos {
		return nil, fmt.Errorf("photo index out of range")
	}
	mu.Lock()
	defer mu.Unlock()
	b, err := os.ReadFile(photoPath(siteID, roomID, idx))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func DeletePhoto(siteID, roomID string, idx int) error {
	if idx < 0 || idx >= MaxRoomPhotos {
		return fmt.Errorf("photo index out of range")
	}
	mu.Lock()
	defer mu.Unlock()
	_ = os.Remove(photoPath(siteID, roomID, idx))
	// compact: shift higher indices down
	for i := idx; i < MaxRoomPhotos-1; i++ {
		src := photoPath(siteID, roomID, i+1)
		dst := photoPath(siteID, roomID, i)
		if _, err := os.Stat(src); err == nil {
			_ = os.Rename(src, dst)
		} else {
			_ = os.Remove(dst)
		}
	}
	_ = os.Remove(photoPath(siteID, roomID, MaxRoomPhotos-1))
	if r, ok := getRoomLocked(siteID, roomID); ok {
		r.PhotoCount = countPhotosLocked(siteID, roomID)
		r.Updated = time.Now().Unix()
		raw, _ := json.MarshalIndent(r, "", "  ")
		_ = os.WriteFile(roomJSONPath(siteID, roomID), append(raw, '\n'), 0o600)
	}
	return nil
}
