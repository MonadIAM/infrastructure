package archive

func (job *Job) tables() []table {
	return []table{
		{
			Name:       "audit_log",
			Bucket:     job.settings.Storage.Buckets[0].Name,
			Projection: "* REPLACE (toString(id) AS id, toString(realm) AS realm, toString(actor) AS actor)",
		},
		{
			Name:       "change_log",
			Bucket:     job.settings.Storage.Buckets[1].Name,
			Projection: "* REPLACE (toString(id) AS id, toString(audit_entry) AS audit_entry, toString(entity) AS entity)",
		},
	}
}
