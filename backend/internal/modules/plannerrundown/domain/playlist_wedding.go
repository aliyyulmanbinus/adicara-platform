package domain

// The 30 songs of the wedding playlist (Wedding_Planner.pdf p. 15-16, 06 BE 5.4): titles and artists as in the PDF (two
// spellings of artists corrected); the moment tags are suggestions of Adicara, the PDF has none. "Janji Suci" is in the
// PDF twice (no. 2 and 12) and stays twice. TestWeddingPlaylistMatchesSpec reads the table of the spec.

// WeddingPlaylist is the playlist template of the wedding event type.
var WeddingPlaylist = []TemplateSong{
	{Key: "wedding.lagu_01", Title: "Akad", Artist: "Payung Teduh", MomentTag: "Akad Khidmat"},
	{Key: "wedding.lagu_02", Title: "Janji Suci", Artist: "Yovie & Nuno", MomentTag: "Akad Khidmat"},
	{Key: "wedding.lagu_03", Title: "Teman Hidup", Artist: "Tulus", MomentTag: "Akad Khidmat"},
	{Key: "wedding.lagu_04", Title: "Kesempurnaan Cinta", Artist: "Rizky Febian", MomentTag: "Kirab / Entrance"},
	{Key: "wedding.lagu_05", Title: "Marry You", Artist: "Maroon 5", MomentTag: "Kirab / Entrance"},
	{Key: "wedding.lagu_06", Title: "Kamulah Satu-Satunya", Artist: "Dewa 19", MomentTag: "Kirab / Entrance"},
	{Key: "wedding.lagu_07", Title: "Cinta Kan Membawamu", Artist: "Dewa 19", MomentTag: "Prosesi Adat"},
	{Key: "wedding.lagu_08", Title: "Aku Milikmu", Artist: "Ari Lasso", MomentTag: "Prosesi Adat"},
	{Key: "wedding.lagu_09", Title: "Anugrah Terindah Yang Pernah Kumiliki", Artist: "Sheila On 7", MomentTag: "Prosesi Adat"},
	{Key: "wedding.lagu_10", Title: "Kala Cinta Menggoda", Artist: "Chrisye", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_11", Title: "Cantik", Artist: "Kahitna", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_12", Title: "Janji Suci", Artist: "Yovie & Nuno", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_13", Title: "Jatuh Hati", Artist: "Raisa", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_14", Title: "Cinta Luar Biasa", Artist: "Andmesh Kamaleng", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_15", Title: "Sugar", Artist: "Maroon 5", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_16", Title: "Just The Way You Are", Artist: "Bruno Mars", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_17", Title: "Kau Adalah", Artist: "Isyana Sarasvati", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_18", Title: "Anything You Want", Artist: "Reality Club", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_19", Title: "Lebih Indah", Artist: "Adera", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_20", Title: "You Are The Reason", Artist: "Calum Scott", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_21", Title: "Thinking Out Loud", Artist: "Ed Sheeran", MomentTag: "Kirab / Entrance"},
	{Key: "wedding.lagu_22", Title: "Perfect", Artist: "Ed Sheeran", MomentTag: "Kirab / Entrance"},
	{Key: "wedding.lagu_23", Title: "Mine", Artist: "Petra Sihombing", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_24", Title: "Asmalibrasi", Artist: "Soegi Bornean", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_25", Title: "Komang", Artist: "Raim Laode", MomentTag: "Santap & Ramah Tamah"},
	{Key: "wedding.lagu_26", Title: "Untukku", Artist: "Chrisye", MomentTag: "Penutup"},
	{Key: "wedding.lagu_27", Title: "Sempurna", Artist: "Andra and The Backbone", MomentTag: "Penutup"},
	{Key: "wedding.lagu_28", Title: "Shallow", Artist: "Lady Gaga", MomentTag: "Games & Lempar Bunga"},
	{Key: "wedding.lagu_29", Title: "A Piece Of You", Artist: "Nathaniel Constantin", MomentTag: "Penutup"},
	{Key: "wedding.lagu_30", Title: "Last Night On Earth", Artist: "Green Day", MomentTag: "Penutup"},
}
