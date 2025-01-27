package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/models"
)

type queryable interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type executable interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

const (
	sqlFlavor = sqlbuilder.PostgreSQL
)

// Ensure Repository satisfies the interface
var _ models.Repository = &Repository{}

type Repository struct {
	queryable
	executable
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db, db}
}

func NewRepositoryTx(tx *sql.Tx) *Repository {
	return &Repository{tx, tx}
}

func (r *Repository) CreateACO(ctx context.Context, aco models.ACO) error {
	ib := sqlFlavor.NewInsertBuilder().InsertInto("acos")
	ib.Cols("uuid", "cms_id", "client_id", "name", "blacklisted",
		"termination_details")
	ib.Values(aco.UUID, aco.CMSID, aco.ClientID, aco.Name, aco.Blacklisted,
		termination{aco.TerminationDetails})
	query, args := ib.Build()
	_, err := r.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) GetACOByUUID(ctx context.Context, uuid uuid.UUID) (*models.ACO, error) {
	return r.getACO(ctx, "uuid", uuid)
}
func (r *Repository) GetACOByClientID(ctx context.Context, clientID string) (*models.ACO, error) {
	return r.getACO(ctx, "client_id", clientID)
}
func (r *Repository) GetACOByCMSID(ctx context.Context, cmsID string) (*models.ACO, error) {
	return r.getACO(ctx, "cms_id", cmsID)
}

func (r *Repository) UpdateACO(ctx context.Context, acoUUID uuid.UUID, fieldsAndValues map[string]interface{}) error {
	ub := sqlFlavor.NewUpdateBuilder().Update("acos")
	for field, value := range fieldsAndValues {
		ub.SetMore(ub.Assign(field, value))
	}
	ub.Where(ub.Equal("uuid", acoUUID))

	query, args := ub.Build()
	result, err := r.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("ACO %s not updated, no row found", acoUUID)
	}

	return nil
}

func (r *Repository) GetLatestCCLFFile(ctx context.Context, cmsID string, cclfNum int, importStatus string, lowerBound, upperBound time.Time, fileType models.CCLFFileType) (*models.CCLFFile, error) {
	sb := sqlFlavor.NewSelectBuilder()
	sb.Select("id", "name", "timestamp", "performance_year")
	sb.From("cclf_files")
	sb.Where(
		sb.Equal("aco_cms_id", cmsID),
		sb.Equal("cclf_num", cclfNum),
		sb.Equal("import_status", importStatus),
		sb.Equal("type", fileType),
	)

	cclfFile := models.CCLFFile{
		ACOCMSID:     cmsID,
		CCLFNum:      cclfNum,
		ImportStatus: importStatus,
		Type:         fileType,
	}

	if !lowerBound.IsZero() && upperBound.IsZero() {
		sb.Where(sb.GreaterEqualThan("timestamp", lowerBound))
	} else if lowerBound.IsZero() && !upperBound.IsZero() {
		sb.Where(sb.LessEqualThan("timestamp", upperBound))
	} else if !lowerBound.IsZero() && !upperBound.IsZero() {
		sb.Where(
			sb.GreaterEqualThan("timestamp", lowerBound),
			sb.LessEqualThan("timestamp", upperBound),
		)
	}
	sb.OrderBy("timestamp").Desc().Limit(1)

	query, args := sb.Build()
	row := r.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&cclfFile.ID, &cclfFile.Name, &cclfFile.Timestamp, &cclfFile.PerformanceYear); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &cclfFile, nil
}

func (r *Repository) CreateCCLFFile(ctx context.Context, cclfFile models.CCLFFile) (uint, error) {
	ib := sqlFlavor.NewInsertBuilder().InsertInto("cclf_files")
	ib.Cols("cclf_num", "name", "aco_cms_id", "timestamp", "performance_year", "import_status", "type").
		Values(cclfFile.CCLFNum, cclfFile.Name, cclfFile.ACOCMSID, cclfFile.Timestamp, cclfFile.PerformanceYear,
			cclfFile.ImportStatus, cclfFile.Type)
	query, args := ib.Build()
	// Append the RETURNING id to retrieve the auto-generated ID value associated with the CCLF File
	query = fmt.Sprintf("%s RETURNING id", query)

	var id uint
	if err := r.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (r *Repository) UpdateCCLFFileImportStatus(ctx context.Context, fileID uint, importStatus string) error {
	ub := sqlFlavor.NewUpdateBuilder().Update("cclf_files")
	ub.Set(ub.Assign("import_status", importStatus))
	ub.Where(ub.Equal("id", fileID))

	query, args := ub.Build()

	result, err := r.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("failed to update file entry %d status to %s, no entry found", fileID, importStatus)
	}

	return nil
}

func (r *Repository) GetCCLFBeneficiaryMBIs(ctx context.Context, cclfFileID uint) ([]string, error) {
	var mbis []string

	sb := sqlFlavor.NewSelectBuilder().Select("mbi").From("cclf_beneficiaries")
	sb.Where(sb.Equal("file_id", cclfFileID))

	query, args := sb.Build()
	rows, err := r.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mbi string
		if err = rows.Scan(&mbi); err != nil {
			return nil, err
		}
		mbis = append(mbis, mbi)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return mbis, nil
}

func (r *Repository) GetCCLFBeneficiaries(ctx context.Context, cclfFileID uint, ignoredMBIs []string) ([]*models.CCLFBeneficiary, error) {
	var beneficiaries []*models.CCLFBeneficiary

	// Subquery to deal with duplicate MBIs found within a single CCLF file.
	// NOTE: We no longer have duplicate MBIs after this PR: https://github.com/CMSgov/bcda-app/pull/583
	// We have to remove duplicates on older files, but once that's done, we can remove the subquery
	// and query for the benes by file_id directly.
	subSB := sqlFlavor.NewSelectBuilder()
	subSB.Select("MAX(id)").From("cclf_beneficiaries").Where(
		subSB.Equal("file_id", cclfFileID),
	).GroupBy("mbi")

	sb := sqlFlavor.NewSelectBuilder()
	sb.Select("id", "file_id", "mbi", "blue_button_id")
	sb.From("cclf_beneficiaries").Where(sb.In("id", subSB))

	if len(ignoredMBIs) != 0 {
		ignored := make([]interface{}, len(ignoredMBIs))
		for i, v := range ignoredMBIs {
			ignored[i] = v
		}
		sb.Where(sb.NotIn("mbi", ignored...))
	}

	query, args := sb.Build()
	rows, err := r.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			bene models.CCLFBeneficiary
			bbID sql.NullString
		)
		if err := rows.Scan(&bene.ID, &bene.FileID, &bene.MBI, &bbID); err != nil {
			return nil, err
		}
		bene.BlueButtonID = bbID.String
		beneficiaries = append(beneficiaries, &bene)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return beneficiaries, nil
}

func (r *Repository) CreateSuppression(ctx context.Context, suppression models.Suppression) error {
	ib := sqlFlavor.NewInsertBuilder().InsertInto("suppressions").
		Cols("file_id", "mbi", "source_code", "effective_date", "preference_indicator",
			"samhsa_source_code", "samhsa_effective_date", "samhsa_preference_indicator",
			"beneficiary_link_key", "aco_cms_id").
		Values(suppression.FileID, suppression.MBI, suppression.SourceCode, suppression.EffectiveDt, suppression.PrefIndicator,
			suppression.SAMHSASourceCode, suppression.SAMHSAEffectiveDt, suppression.SAMHSAPrefIndicator,
			suppression.BeneficiaryLinkKey, suppression.ACOCMSID)
	query, args := ib.Build()

	_, err := r.ExecContext(ctx, query, args...)
	return err
}

func (r *Repository) GetSuppressedMBIs(ctx context.Context, lookbackDays int, upperBound time.Time) ([]string, error) {
	var suppressedMBIs []string

	lookbackDuration := time.Duration(-1*lookbackDays*24) * time.Hour
	lowerBound := upperBound.Add(lookbackDuration)

	subSB := sqlFlavor.NewSelectBuilder()
	subSB.Select("mbi", "MAX(effective_date) as max_date").From("suppressions")
	subSB.Where(
		subSB.GreaterEqualThan("effective_date", lowerBound), subSB.LessEqualThan("effective_date", upperBound),
		subSB.NotEqual("preference_indicator", ""),
	).GroupBy("mbi")

	sb := sqlFlavor.NewSelectBuilder().Distinct().Select("s.mbi")
	sb.From(sb.BuilderAs(subSB, "h")).Join("suppressions s", "s.mbi = h.mbi", "s.effective_date = h.max_date")
	sb.Where(sb.Equal("preference_indicator", "N"))

	query, args := sb.Build()
	rows, err := r.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mbi string
		if err = rows.Scan(&mbi); err != nil {
			return nil, err
		}
		suppressedMBIs = append(suppressedMBIs, mbi)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return suppressedMBIs, nil
}

func (r *Repository) CreateSuppressionFile(ctx context.Context, suppressionFile models.SuppressionFile) (uint, error) {
	ib := sqlFlavor.NewInsertBuilder().InsertInto("suppression_files")
	ib.Cols("name", "timestamp", "import_status").
		Values(suppressionFile.Name, suppressionFile.Timestamp, suppressionFile.ImportStatus)
	query, args := ib.Build()
	// Append the RETURNING id to retrieve the auto-generated ID value associated with the suppression file
	query = fmt.Sprintf("%s RETURNING id", query)
	var id uint
	if err := r.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (r *Repository) UpdateSuppressionFileImportStatus(ctx context.Context, fileID uint, importStatus string) error {
	ub := sqlFlavor.NewUpdateBuilder().Update("suppression_files")
	ub.Set(ub.Assign("import_status", importStatus))
	ub.Where(ub.Equal("id", fileID))

	query, args := ub.Build()
	result, err := r.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return fmt.Errorf("SuppressionFile %d not updated, no row found", fileID)
	}

	return nil
}

var jobColumns []string = []string{"id", "aco_id", "request_url", "status", "transaction_time", "job_count", "completed_job_count", "created_at", "updated_at"}

func (r *Repository) GetJobs(ctx context.Context, acoID uuid.UUID, statuses ...models.JobStatus) ([]*models.Job, error) {
	s := make([]interface{}, len(statuses))
	for i, v := range statuses {
		s[i] = v
	}

	sb := sqlFlavor.NewSelectBuilder()
	sb.Select(jobColumns...)
	sb.From("jobs").Where(
		sb.Equal("aco_id", acoID),
	)

	if len(s) > 0 {
		sb.Where(sb.In("status", s...))
	}

	query, args := sb.Build()
	return r.getJobs(ctx, query, args...)

}

func (r *Repository) GetJobsByUpdateTimeAndStatus(ctx context.Context, lowerBound, upperBound time.Time, statuses ...models.JobStatus) ([]*models.Job, error) {
	s := make([]interface{}, len(statuses))
	for i, v := range statuses {
		s[i] = v
	}

	sb := sqlFlavor.NewSelectBuilder().Select(jobColumns...).From("jobs")
	if !lowerBound.IsZero() {
		sb.Where(sb.GreaterEqualThan("updated_at", lowerBound))
	}
	if !upperBound.IsZero() {
		sb.Where(sb.LessEqualThan("updated_at", upperBound))
	}

	if len(s) > 0 {
		sb.Where(sb.In("status", s...))
	}

	query, args := sb.Build()
	return r.getJobs(ctx, query, args...)
}

func (r *Repository) GetJobByID(ctx context.Context, jobID uint) (*models.Job, error) {
	sb := sqlFlavor.NewSelectBuilder()
	sb.Select(jobColumns...)
	sb.From("jobs").Where(sb.Equal("id", jobID))

	query, args := sb.Build()

	var (
		j                                     models.Job
		transactionTime, createdAt, updatedAt sql.NullTime
	)

	err := r.QueryRowContext(ctx, query, args...).Scan(&j.ID, &j.ACOID, &j.RequestURL, &j.Status, &transactionTime,
		&j.JobCount, &j.CompletedJobCount, &createdAt, &updatedAt)
	j.TransactionTime, j.CreatedAt, j.UpdatedAt = transactionTime.Time, createdAt.Time, updatedAt.Time

	if err != nil {
		return nil, err
	}

	return &j, nil
}

func (r *Repository) CreateJob(ctx context.Context, j models.Job) (uint, error) {
	// User raw builder since we need to retrieve the associated ID
	ib := sqlFlavor.NewInsertBuilder().InsertInto("jobs")
	ib.Cols("aco_id", "request_url", "status",
		"transaction_time", "job_count", "completed_job_count",
		"created_at", "updated_at").
		Values(j.ACOID, j.RequestURL, j.Status,
			j.TransactionTime, j.JobCount, j.CompletedJobCount,
			sqlbuilder.Raw("NOW()"), sqlbuilder.Raw("NOW()"))

	query, args := ib.Build()
	// Append the RETURNING id to retrieve the auto-generated ID value associated with the Job
	query = fmt.Sprintf("%s RETURNING id", query)

	var id uint
	if err := r.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (r *Repository) UpdateJob(ctx context.Context, j models.Job) error {
	ub := sqlFlavor.NewUpdateBuilder().Update("jobs")
	ub.Set(
		ub.Assign("aco_id", j.ACOID),
		ub.Assign("request_url", j.RequestURL),
		ub.Assign("status", j.Status),
		ub.Assign("transaction_time", j.TransactionTime),
		ub.Assign("job_count", j.JobCount),
		ub.Assign("completed_job_count", j.CompletedJobCount),
		ub.Assign("updated_at", sqlbuilder.Raw("NOW()")),
	)
	ub.Where(ub.Equal("id", j.ID))
	query, args := ub.Build()

	res, err := r.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows != 1 {
		return fmt.Errorf("expected to affect 1 row, affected %d", rows)
	}

	return nil
}

func (r *Repository) GetJobKeys(ctx context.Context, jobID uint) ([]*models.JobKey, error) {
	sb := sqlFlavor.NewSelectBuilder().Select("id", "file_name", "resource_type").From("job_keys")
	sb.Where(sb.Equal("job_id", jobID))

	query, args := sb.Build()
	rows, err := r.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*models.JobKey
	for rows.Next() {
		jk := models.JobKey{JobID: jobID}
		if err = rows.Scan(&jk.ID, &jk.FileName, &jk.ResourceType); err != nil {
			return nil, err
		}
		keys = append(keys, &jk)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

func (r *Repository) getJobs(ctx context.Context, query string, args ...interface{}) ([]*models.Job, error) {
	rows, err := r.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		jobs                                  []*models.Job
		transactionTime, createdAt, updatedAt sql.NullTime
	)
	for rows.Next() {
		var j models.Job
		if err = rows.Scan(&j.ID, &j.ACOID, &j.RequestURL, &j.Status, &transactionTime,
			&j.JobCount, &j.CompletedJobCount, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		j.TransactionTime, j.CreatedAt, j.UpdatedAt = transactionTime.Time, createdAt.Time, updatedAt.Time
		jobs = append(jobs, &j)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *Repository) getACO(ctx context.Context, field string, value interface{}) (*models.ACO, error) {
	sb := sqlFlavor.NewSelectBuilder().Select("id", "uuid", "cms_id", "name",
		"client_id", "group_id", "system_id", "alpha_secret", "public_key",
		"blacklisted", "termination_details").From("acos")
	sb.Where(sb.Equal(field, value))

	query, args := sb.Build()
	row := r.QueryRowContext(ctx, query, args...)
	var (
		aco                                                              models.ACO
		termination                                                      termination
		name, cmsID, clientID, alphaSecret, publicKey, groupID, systemID sql.NullString
	)
	err := row.Scan(&aco.ID, &aco.UUID, &cmsID, &name,
		&clientID, &groupID, &systemID, &alphaSecret,
		&publicKey, &aco.Blacklisted, &termination)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no ACO record found for %s", value)
		}
		return nil, err
	}
	aco.Name, aco.ClientID, aco.AlphaSecret = name.String, clientID.String, alphaSecret.String
	aco.PublicKey, aco.GroupID, aco.SystemID = publicKey.String, groupID.String, systemID.String
	aco.CMSID = &cmsID.String
	aco.TerminationDetails = termination.Termination
	return &aco, nil
}
