package api

// computePageBounds returns safe zero-based slice bounds and page values.
func computePageBounds(count, page, itemsOnPage int) (from, to, safePage, totalPages int) {
	if count <= 0 || itemsOnPage <= 0 {
		return 0, 0, 0, 0
	}
	totalPages = 1 + (count-1)/itemsOnPage
	if page < 0 {
		page = 0
	} else if page >= totalPages {
		page = totalPages - 1
	}
	from = page * itemsOnPage
	if itemsOnPage > count-from {
		to = count
	} else {
		to = from + itemsOnPage
	}
	return from, to, page, totalPages
}
