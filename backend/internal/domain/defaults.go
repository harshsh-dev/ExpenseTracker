package domain

// DefaultCategories seed the store on first run. Users can edit them afterward.
func DefaultCategories() []Category {
	seed := []struct {
		name string
		col  string
		subs []string
	}{
		{"Food & Dining", "#ef4444", []string{"Restaurants", "Cafes", "Takeout"}},
		{"Groceries", "#f97316", []string{"Supermarket", "Vegetables", "Household"}},
		{"Housing", "#f59e0b", []string{"Rent", "Maintenance", "Repairs"}},
		{"Utilities", "#eab308", []string{"Electricity", "Water", "Gas", "Internet", "Mobile"}},
		{"Transportation", "#84cc16", []string{"Fuel", "Cab/Ride-share", "Public transit", "Parking"}},
		{"Health & Medical", "#22c55e", []string{"Doctor", "Pharmacy", "Insurance premium"}},
		{"Shopping", "#10b981", []string{"Clothing", "Electronics", "Home"}},
		{"Entertainment", "#14b8a6", []string{"Streaming", "Movies", "Games", "Events"}},
		{"Subscriptions", "#06b6d4", []string{"Software", "Memberships"}},
		{"Education", "#0ea5e9", []string{"Courses", "Books", "Fees"}},
		{"Personal Care", "#3b82f6", []string{"Salon", "Gym", "Cosmetics"}},
		{"Travel", "#6366f1", []string{"Flights", "Hotels", "Holidays"}},
		{"EMI / Loans", "#8b5cf6", []string{"Home loan", "Car loan", "Credit card"}},
		{"Insurance", "#a855f7", []string{"Life", "Health", "Vehicle"}},
		{"Gifts & Donations", "#d946ef", []string{"Gifts", "Charity"}},
		{"Taxes & Fees", "#ec4899", []string{"Income tax", "Bank fees"}},
		{"Miscellaneous", "#64748b", []string{"Uncategorized"}},
	}

	cats := make([]Category, 0, len(seed))
	for _, s := range seed {
		cats = append(cats, Category{
			Name:          s.name,
			Color:         s.col,
			Subcategories: s.subs,
		})
	}
	return cats
}
