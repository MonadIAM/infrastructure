package archive

func (job *Job) tables() []table {
	return []table{
		{
			Name:   "audit_log",
			Bucket: job.settings.Storage.Buckets[0].Name,
		},
		{
			Name:   "change_log",
			Bucket: job.settings.Storage.Buckets[1].Name,
		},
	}
}
