package domain

// The 37 items of the seserahan list of Wedding_Planner.pdf p. 9-10 (04 BE 5.3): names, categories and reference prices
// are the PDF's. The purchase links of the PDF are the private links of its author and are not copied. Done marks the
// items the PDF shows as "Selesai"; only the sample project uses it, the template starts every item at "belum".

import pd "github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/planner/domain"

// TemplateItem is one item of the shopping template.
type TemplateItem struct {
	Category string
	Name     string
	Price    int64 // reference price per unit; the quantity is 1
	Done     bool  // "Selesai" in the PDF
}

// WeddingShopping is the shopping template of the wedding event type: 37 items, Rp 4.578.500 (16 done = 2.063.800).
var WeddingShopping = []TemplateItem{
	{"Perangkat Alat Solat", "Mukena", 194000, true},
	{"Perangkat Alat Solat", "Al-Quran", 70500, false},
	{"Perangkat Alat Solat", "Sajadah Couple", 233000, true},
	{"Perangkat Alat Solat", "Tasbih Digital", 99800, false},
	{"Make Up & Skin Care", "Cushion", 175700, true},
	{"Make Up & Skin Care", "Cream Blush", 29000, false},
	{"Make Up & Skin Care", "Lipmatte", 79000, true},
	{"Make Up & Skin Care", "Liptint", 75400, false},
	{"Make Up & Skin Care", "Loose Powder", 73200, true},
	{"Make Up & Skin Care", "Mascara", 66700, false},
	{"Make Up & Skin Care", "Eyebrow", 61300, true},
	{"Make Up & Skin Care", "Eyeliner", 34900, false},
	{"Make Up & Skin Care", "Micellar Water", 44500, true},
	{"Make Up & Skin Care", "Facial Wash", 53800, false},
	{"Make Up & Skin Care", "Moisturizer", 201600, true},
	{"Make Up & Skin Care", "Sunscreen", 162800, false},
	{"Make Up & Skin Care", "Serum", 95400, false},
	{"Make Up & Skin Care", "Night Cream", 115600, true},
	{"Peralatan Mandi", "Sabun Mandi", 195000, false},
	{"Peralatan Mandi", "Shampo", 195000, true},
	{"Peralatan Mandi", "Sikat Gigi", 67000, false},
	{"Peralatan Mandi", "Body Lotion", 40700, true},
	{"Peralatan Mandi", "Body Scrub", 54100, false},
	{"Peralatan Mandi", "Body Serum", 65300, true},
	{"Peralatan Mandi", "Sisir (Wet Brush)", 77600, false},
	{"Peralatan Mandi", "Handuk Couple", 296000, true},
	{"Pakaian Dalam", "Bra", 152500, false},
	{"Pakaian Dalam", "Celana Dalam", 72600, true},
	{"Pakaian Dalam", "Lingerie", 133000, false},
	{"Pakaian Dalam", "Baju Tidur", 159300, true},
	{"Pakaian Luar", "Gamis/Baju Couple", 388800, false},
	{"Pakaian Luar", "Hijab/Khimar", 57000, true},
	{"Pakaian Luar", "Sepatu/Flat Shoes", 116900, false},
	{"Pakaian Luar", "Dompet/Tas Kecil (Sling Bag)", 149300, false},
	{"Pakaian Luar", "Setelan Kebaya Muslimah", 108200, false},
	{"Kebutuhan Lain", "Seprai Bed Cover", 346100, false},
	{"Kebutuhan Lain", "Box Seserahan (10 Kotak)", 37900, false},
}

const weddingTemplatePrefix = "wedding.seserahan_"

// Key is the template_key of the item: the prefix and the name in lower-case underscore form, not cut (37 unique keys,
// the longest 44 characters), so a template item stays recognisable when the user renames it.
func (t TemplateItem) Key() string { return weddingTemplatePrefix + underscoreKey(t.Name) }

// TemplateFor is the shopping template of an event type; only the wedding has one (the others have categories only).
func TemplateFor(t pd.EventType) []TemplateItem {
	if t == pd.EventWedding {
		return WeddingShopping
	}

	return nil
}
