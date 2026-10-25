package domain

// The event type registry is code, not data: it is the one source for icons,
// labels, default sessions, metadata fields and which templates a type owns.
// The CHECK on planner_projects.event_type must list exactly these keys
// (guarded by a test that reads pg_get_constraintdef).
//
// Fields owned by the other planner modules (audio moment tags) are added here
// when those modules are built; their data is in
// PLANNER_SPEC/00_KONTRAK_BERSAMA/TEMPLATE_EVENT.md. Menu 02 added the document
// subjects, the legal widget title and the bureaucracy guide (TEMPLATE_EVENT
// 4.1); menu 03 added the contributor roles (TEMPLATE_EVENT 4); menu 04 added
// the vendor and shopping categories and the shopping rollup line (vendorcatalog.go);
// menu 05 added the guest sides and categories (guestcatalog.go, TEMPLATE_EVENT 4.2).

const (
	EventWedding   EventType = "wedding"
	EventKhitanan  EventType = "khitanan"
	EventAqiqah    EventType = "aqiqah"
	EventPeresmian EventType = "peresmian"
	EventLaunching EventType = "launching"
	EventBirthday  EventType = "birthday"
	EventWisuda    EventType = "wisuda"
	EventSeminar   EventType = "seminar"
	EventOther     EventType = "other"
)

// SessionSeed is a default session created with every project of the type.
// ScopeKey is explicit rather than derived from Name: the wedding session
// "Akad Nikah" is keyed `akad` everywhere in the specs, but the derivation
// rule used for user-made sessions would give `akad_nikah`.
type SessionSeed struct {
	ScopeKey string `json:"scope_key"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	SameDay  bool   `json:"same_day"` // born with session_date = event_date
}

type FieldType string

const (
	FieldString FieldType = "string"
	FieldInt    FieldType = "int"
	FieldDate   FieldType = "date"
)

// FieldSpec describes one key of planner_projects.metadata.
type FieldSpec struct {
	Key      string    `json:"key"`
	Label    string    `json:"label"`
	Type     FieldType `json:"type"`
	MaxLen   int       `json:"max_len,omitempty"` // string: runes
	Min      *int      `json:"min,omitempty"`     // int
	Max      *int      `json:"max,omitempty"`
	Required bool      `json:"required"`
}

// Option is a key with its display label (document subjects, contributor roles, guest sides and categories).
type Option struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// VendorCategory is one entry of the vendor checklist of an event type (menu 04).
// BudgetItemKeys are the item_keys of the budget lines and scenario items that
// hold its cost (TEMPLATE_EVENT 15.2, 04 BE 5.1); empty = no line, only a manual
// limit. They stay on the server.
type VendorCategory struct {
	Key            string   `json:"key"`
	Label          string   `json:"label"`
	BudgetItemKeys []string `json:"-"`
}

// ShoppingRollup is the budget line the shopping list is rolled up into by
// default (TEMPLATE_EVENT 15.4). It is looked up by Key in any session; when the
// project has no such line it is created with Name and Category.
type ShoppingRollup struct{ Key, Name, Category string }

// Guide is the callout above the document list: a title and exactly four steps.
type Guide struct {
	Title string   `json:"title"`
	Steps []string `json:"steps"`
}

// Labels are the words in the UI that depend on the event type.
type Labels struct {
	Guest     string `json:"guest"`      // noun for a guest: "Tamu", seminar "Peserta"
	GuestMenu string `json:"guest_menu"` // menu and widget title
}

// TemplateSet is what the type owns; GET /event-types reduces it to
// templates_available by the appliers that are actually registered.
type TemplateSet struct {
	TimelinePresets     []string
	IncludeAdicaraTasks bool
	AdminDocs           bool
	BudgetLines         bool
	ShoppingList        bool
	Rundown             bool
	Playlist            bool
}

type EventTypeSpec struct {
	Key            EventType     `json:"key"`
	Label          string        `json:"label"`
	Icon           string        `json:"icon"`
	LeadTimeDays   int           `json:"lead_time_days"` // standard horizon of the timeline template
	Sessions       []SessionSeed `json:"sessions"`
	MetadataFields []FieldSpec   `json:"metadata_fields"`
	Labels         Labels        `json:"labels"`
	Templates      TemplateSet   `json:"-"`

	// Owned by menu 02 (documents). AdminLabel is the title of the dashboard
	// widget and is read by the admin_docs provider; it is not sent to clients.
	AdminSubjects []Option `json:"admin_subjects"`
	AdminGuide    Guide    `json:"admin_guide"`
	AdminLabel    string   `json:"-"`

	// Owned by menu 03 (finance): the roles a savings contributor can have.
	// Every type lists "other"; it is the default.
	ContributorRoles []Option `json:"contributor_roles"`

	// Owned by menu 04 (vendors and shopping). ShoppingCategories is plain text,
	// stored as typed in planner_shopping_items.category.
	VendorCategories   []VendorCategory `json:"vendor_categories"`
	ShoppingCategories []string         `json:"shopping_categories"`
	ShoppingRollup     ShoppingRollup   `json:"-"`

	// Owned by menu 05 (guests). No sides = the guest list is not split (birthday, wisuda, other). Any other text a
	// user types as a category is a custom category; the fallback "lainnya" is not part of the registry.
	GuestSides      []Option `json:"guest_sides"`
	GuestCategories []Option `json:"guest_categories"`
}

// TemplatesAvailable is the part of TemplateSet that can be requested now.
type TemplatesAvailable struct {
	TimelinePresets     []string `json:"timeline_presets"`
	IncludeAdicaraTasks bool     `json:"include_adicara_tasks"`
	AdminDocs           bool     `json:"admin_docs"`
	BudgetLines         bool     `json:"budget_lines"`
	ShoppingList        bool     `json:"shopping_list"`
	Rundown             bool     `json:"rundown"`
	Playlist            bool     `json:"playlist"`
}

// Available reduces the set to what has an applier. has reports whether an
// applier is registered for a template key.
func (t TemplateSet) Available(has func(key string) bool) TemplatesAvailable {
	out := TemplatesAvailable{TimelinePresets: []string{}}
	if has(TemplateTimeline) {
		out.TimelinePresets = append(out.TimelinePresets, t.TimelinePresets...)
		out.IncludeAdicaraTasks = t.IncludeAdicaraTasks
	}
	out.AdminDocs = t.AdminDocs && has(TemplateAdminDocs)
	out.BudgetLines = t.BudgetLines && has(TemplateBudget)
	out.ShoppingList = t.ShoppingList && has(TemplateShopping)
	out.Rundown = t.Rundown && has(TemplateRundown)
	out.Playlist = t.Playlist && has(TemplatePlaylist)

	return out
}

// Registry is an ordered, immutable list of event types.
type Registry struct {
	specs []EventTypeSpec
	index map[EventType]int
}

func NewRegistry(specs []EventTypeSpec) *Registry {
	r := &Registry{specs: specs, index: make(map[EventType]int, len(specs))}
	for i, s := range specs {
		r.index[s.Key] = i
	}

	return r
}

func (r *Registry) Get(key EventType) (EventTypeSpec, bool) {
	i, ok := r.index[key]
	if !ok {
		return EventTypeSpec{}, false
	}

	return r.specs[i], true
}

func (r *Registry) All() []EventTypeSpec { return r.specs }

func (r *Registry) Keys() []string {
	keys := make([]string, len(r.specs))
	for i, s := range r.specs {
		keys[i] = string(s.Key)
	}

	return keys
}

func intp(v int) *int { return &v }

func str(key, label string, maxLen int, required bool) FieldSpec {
	return FieldSpec{Key: key, Label: label, Type: FieldString, MaxLen: maxLen, Required: required}
}

// commonMetadata is accepted by every event type.
func commonMetadata() []FieldSpec {
	return []FieldSpec{
		str("hero_title", "Judul Hero", 60, false),
		str("venue_name", "Nama Venue", 150, false),
		str("city", "Kota", 80, false),
	}
}

func fields(extra ...FieldSpec) []FieldSpec {
	return append(extra, commonMetadata()...)
}

var defaultLabels = Labels{Guest: "Tamu", GuestMenu: "Buku Tamu & RSVP"}

// weddingAdminSubjects follows Wedding_Planner.pdf p. 11 (the PDF writes CMP/CMW
// for the parents' side; Adicara uses ortu_cpp/ortu_cpw).
var weddingAdminSubjects = []Option{
	{"cpp", "CPP"}, {"cpw", "CPW"}, {"ortu_cpp", "Orang Tua CPP"}, {"ortu_cpw", "Orang Tua CPW"}, {"bersama", "Bersama"},
}

// weddingAdminGuide is the note of Wedding_Planner.pdf p. 11 (forms N1, N2, N4).
var weddingAdminGuide = Guide{
	Title: "Panduan Pengurusan Berkas Nikah",
	Steps: []string{
		"RT & RW: minta surat pengantar dengan membawa fotokopi KTP kamu dan pasangan serta fotokopi Kartu Keluarga.",
		"Kelurahan: bawa surat pengantar untuk mengisi formulir N1, N2, N4 dan surat keterangan belum menikah; syarat pasfoto 3×4 dan 2×3 masing-masing 2 lembar, fotokopi KTP CPW dan CPP 2 lembar, fotokopi KK CPP dan CPW 2 lembar, dan surat pengantar RT/RW.",
		"KUA asal (numpang nikah): minta surat rekomendasi numpang nikah di tempat tujuan; siapkan fotokopi KTP CPP dan CPW, fotokopi KK, surat pengantar kelurahan (N1, N2, N4), pasfoto 2×3 dan 3×4 masing-masing 2 lembar.",
		"KUA tempat akad: serahkan surat rekomendasi pindah nikah dari KUA asal, fotokopi KTP dan KK, foto berwarna 2×3 dan 3×4 masing-masing 2 lembar, fotokopi ijazah terakhir, dan fotokopi akta kelahiran.",
	},
}

// contributorRoleLabels are the Indonesian labels of the role keys; a type
// picks its own subset (TEMPLATE_EVENT 4).
var contributorRoleLabels = map[string]string{
	"groom": "Mempelai Pria", "bride": "Mempelai Wanita", "host": "Tuan Rumah", "internal": "Internal/Panitia",
	"family": "Keluarga", "sponsor": "Sponsor", "other": "Lainnya",
}

var contributorRoleKeys = map[EventType][]string{
	EventWedding:   {"groom", "bride", "family", "sponsor", "other"},
	EventKhitanan:  {"host", "family", "other"},
	EventAqiqah:    {"host", "family", "other"},
	EventPeresmian: {"internal", "sponsor", "other"},
	EventLaunching: {"internal", "sponsor", "other"},
	EventBirthday:  {"host", "family", "other"},
	EventWisuda:    {"internal", "family", "sponsor", "other"},
	EventSeminar:   {"internal", "sponsor", "other"},
	EventOther:     {"host", "internal", "sponsor", "other"},
}

func contributorRolesFor(t EventType) []Option {
	keys := contributorRoleKeys[t]
	out := make([]Option, len(keys))
	for i, k := range keys {
		out[i] = Option{Key: k, Label: contributorRoleLabels[k]}
	}

	return out
}

// The eight other types share subjects and a permit guide. The guide is an
// Adicara proposal, not from the PDF; requirements differ per region, so the
// text sends the user to the local office.
var generalAdminSubjects = []Option{{"panitia", "Panitia"}, {"lembaga", "Lembaga/Instansi"}, {"umum", "Umum"}}

var generalAdminGuide = Guide{
	Title: "Panduan Perizinan Acara",
	Steps: []string{
		"RT/RW: sampaikan rencana acara dan minta surat pengantar atau persetujuan lingkungan, terutama bila acara memakai jalan, tenda, atau pengeras suara.",
		"Kelurahan dan Polsek: sampaikan pemberitahuan acara dengan membawa surat pengantar RT/RW dan fotokopi KTP penanggung jawab; tanyakan apakah acara Anda memerlukan izin keramaian.",
		"Izin keramaian (bila diperlukan): untuk acara yang ramai, memakai jalan umum, atau menghadirkan hiburan dan musik, ajukan ke Polsek setempat dengan rincian acara; tanyakan syarat dan biaya yang berlaku.",
		"Pengelola venue: pastikan tanggal, jam, kapasitas, aturan lokasi (suara, parkir, dekorasi), dan kontrak atau surat izin penggunaan tempat secara tertulis.",
	},
}

const (
	weddingAdminLabel = "Kelengkapan Berkas KUA"
	generalAdminLabel = "Kelengkapan Perizinan"
)

// generalTemplates is what the eight non-wedding types own (the wedding adds
// shopping and playlist templates; "other" has no document template).
func generalTemplates(adminDocs bool) TemplateSet {
	return TemplateSet{TimelinePresets: []string{"general"}, AdminDocs: adminDocs, BudgetLines: true, Rundown: true}
}

// DefaultRegistry returns the nine event types of Adicara. Wedding comes from
// Wedding_Planner.pdf; the other eight are Adicara proposals.
func DefaultRegistry() *Registry {
	specs := []EventTypeSpec{
		{
			Key: EventWedding, Label: "Pernikahan", Icon: "💍", LeadTimeDays: 180,
			Sessions: []SessionSeed{
				{ScopeKey: "lamaran", Name: "Lamaran", Icon: "💍"}, // usually another day: no date until the user sets one
				{ScopeKey: "akad", Name: "Akad Nikah", Icon: "📜", SameDay: true},
				{ScopeKey: "resepsi", Name: "Resepsi", Icon: "🏛️", SameDay: true},
			},
			MetadataFields: fields(
				str("bride_name", "Nama Mempelai Wanita", 150, true),
				str("groom_name", "Nama Mempelai Pria", 150, true),
				str("bride_parents", "Orang Tua Mempelai Wanita", 150, false),
				str("groom_parents", "Orang Tua Mempelai Pria", 150, false),
			),
			Labels: defaultLabels,
			Templates: TemplateSet{
				TimelinePresets: []string{"wedding_pdf_44", "wedding_essential_22"}, IncludeAdicaraTasks: true,
				AdminDocs: true, BudgetLines: true, ShoppingList: true, Rundown: true, Playlist: true,
			},
			AdminSubjects: weddingAdminSubjects, AdminGuide: weddingAdminGuide, AdminLabel: weddingAdminLabel,
		},
		{
			Key: EventKhitanan, Label: "Khitanan", Icon: "🕌", LeadTimeDays: 45,
			Sessions: []SessionSeed{
				{ScopeKey: "prosesi_doa", Name: "Prosesi Doa", Icon: "🕌", SameDay: true},
				{ScopeKey: "hiburan", Name: "Hiburan", Icon: "🎉", SameDay: true},
				{ScopeKey: "jamuan", Name: "Jamuan", Icon: "🍽️", SameDay: true},
			},
			MetadataFields: fields(str("child_name", "Nama Anak", 150, true), str("parent_names", "Nama Orang Tua", 150, false)),
			Labels:         defaultLabels, Templates: generalTemplates(true),
		},
		{
			Key: EventAqiqah, Label: "Aqiqah", Icon: "🍼", LeadTimeDays: 21,
			Sessions: []SessionSeed{
				{ScopeKey: "prosesi_aqiqah", Name: "Prosesi Aqiqah", Icon: "🕌", SameDay: true},
				{ScopeKey: "syukuran", Name: "Syukuran", Icon: "🍽️", SameDay: true},
			},
			MetadataFields: fields(
				str("baby_name", "Nama Bayi", 150, true),
				FieldSpec{Key: "birth_date", Label: "Tanggal Lahir", Type: FieldDate},
				str("parent_names", "Nama Orang Tua", 150, false),
				FieldSpec{Key: "goat_count", Label: "Jumlah Kambing", Type: FieldInt, Min: intp(0), Max: intp(50)},
			),
			Labels: defaultLabels, Templates: generalTemplates(true),
		},
		{
			Key: EventPeresmian, Label: "Peresmian", Icon: "🏢", LeadTimeDays: 60,
			Sessions: []SessionSeed{
				{ScopeKey: "pembukaan", Name: "Pembukaan", Icon: "🎭", SameDay: true},
				{ScopeKey: "prosesi_peresmian", Name: "Prosesi Peresmian", Icon: "✂️", SameDay: true},
				{ScopeKey: "ramah_tamah", Name: "Ramah Tamah", Icon: "🍽️", SameDay: true},
			},
			MetadataFields: fields(str("organization_name", "Nama Organisasi", 150, true), str("vip_name", "Nama Tamu Kehormatan", 150, false)),
			Labels:         defaultLabels, Templates: generalTemplates(true),
		},
		{
			Key: EventLaunching, Label: "Launching", Icon: "🚀", LeadTimeDays: 90,
			Sessions: []SessionSeed{
				{ScopeKey: "opening", Name: "Opening", Icon: "🎬", SameDay: true},
				{ScopeKey: "peluncuran", Name: "Peluncuran", Icon: "🚀", SameDay: true},
				{ScopeKey: "networking", Name: "Networking", Icon: "🤝", SameDay: true},
			},
			MetadataFields: fields(
				str("brand_name", "Nama Brand", 150, true),
				str("product_name", "Nama Produk", 150, false),
				str("target_media", "Target Media", 150, false),
			),
			Labels: defaultLabels, Templates: generalTemplates(true),
		},
		{
			Key: EventBirthday, Label: "Ulang Tahun", Icon: "🎂", LeadTimeDays: 45,
			Sessions: []SessionSeed{
				{ScopeKey: "welcome", Name: "Welcome", Icon: "🎉", SameDay: true},
				{ScopeKey: "acara_utama", Name: "Acara Utama", Icon: "🎂", SameDay: true},
				{ScopeKey: "after_party", Name: "After Party", Icon: "🎶", SameDay: true},
			},
			MetadataFields: fields(
				str("celebrant_name", "Nama yang Berulang Tahun", 150, true),
				FieldSpec{Key: "age", Label: "Usia", Type: FieldInt, Min: intp(1), Max: intp(120)},
				str("theme", "Tema", 150, false),
			),
			Labels: defaultLabels, Templates: generalTemplates(true),
		},
		{
			Key: EventWisuda, Label: "Wisuda", Icon: "🎓", LeadTimeDays: 60,
			Sessions: []SessionSeed{
				{ScopeKey: "prosesi_senat", Name: "Prosesi Senat", Icon: "📜", SameDay: true},
				{ScopeKey: "pelantikan", Name: "Pelantikan", Icon: "🎓", SameDay: true},
				{ScopeKey: "foto_studio", Name: "Foto Studio", Icon: "📸", SameDay: true},
			},
			MetadataFields: fields(
				str("institution_name", "Nama Institusi", 150, true),
				str("faculty", "Fakultas", 150, false),
				FieldSpec{Key: "graduate_count", Label: "Jumlah Wisudawan", Type: FieldInt, Min: intp(0), Max: intp(maxCount)},
			),
			Labels: defaultLabels, Templates: generalTemplates(true),
		},
		{
			Key: EventSeminar, Label: "Seminar", Icon: "🎤", LeadTimeDays: 75,
			Sessions: []SessionSeed{
				{ScopeKey: "registrasi", Name: "Registrasi", Icon: "📝", SameDay: true},
				{ScopeKey: "sesi_pagi", Name: "Sesi Pagi", Icon: "🎤", SameDay: true},
				{ScopeKey: "coffee_break", Name: "Coffee Break", Icon: "☕", SameDay: true},
				{ScopeKey: "sesi_sore", Name: "Sesi Sore", Icon: "🎤", SameDay: true},
			},
			MetadataFields: fields(
				str("organizer_name", "Nama Penyelenggara", 150, true),
				str("keynote_speaker", "Pembicara Utama", 150, false),
				FieldSpec{Key: "expected_participants", Label: "Perkiraan Peserta", Type: FieldInt, Min: intp(0), Max: intp(maxCount)},
			),
			Labels:    Labels{Guest: "Peserta", GuestMenu: "Peserta & RSVP"},
			Templates: generalTemplates(true),
		},
		{
			Key: EventOther, Label: "Acara Lainnya", Icon: "✨", LeadTimeDays: 60,
			Sessions:       []SessionSeed{{ScopeKey: "sesi_1", Name: "Sesi 1", Icon: "📋", SameDay: true}},
			MetadataFields: fields(str("organizer_name", "Nama Penyelenggara", 150, false)),
			Labels:         defaultLabels, Templates: generalTemplates(false),
		},
	}

	// The eight non-wedding types share the permit subjects, guide and widget title.
	for i := range specs {
		if specs[i].AdminSubjects == nil {
			specs[i].AdminSubjects, specs[i].AdminGuide, specs[i].AdminLabel = generalAdminSubjects, generalAdminGuide, generalAdminLabel
		}
		specs[i].ContributorRoles = contributorRolesFor(specs[i].Key)
		specs[i].VendorCategories = vendorCategoriesFor(specs[i].Key)
		specs[i].ShoppingCategories = shoppingCategoriesFor(specs[i].Key)
		specs[i].ShoppingRollup = shoppingRollupFor(specs[i].Key)
		specs[i].GuestSides = guestSidesFor(specs[i].Key)
		specs[i].GuestCategories = guestCategoriesFor(specs[i].Key)
	}

	return NewRegistry(specs)
}

// maxCount bounds the head-count metadata fields (safe for JavaScript numbers).
const maxCount = 1_000_000
