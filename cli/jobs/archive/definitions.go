package archive

func tableDefinitions(settings Settings) []table {
	return []table{
		{
			Name:       "audit_log",
			Bucket:     settings.Storage.Buckets[0].Name,
			Projection: "* REPLACE (toString(id) AS id, toString(realm) AS realm, toString(actor) AS actor)",
		},
		{
			Name:       "change_log",
			Bucket:     settings.Storage.Buckets[1].Name,
			Projection: "* REPLACE (toString(id) AS id, toString(audit_entry) AS audit_entry, toString(entity) AS entity)",
		},
	}
}
