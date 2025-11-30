package organization

import (
	"database/sql"

	"github.com/21strive/redifu"
	"github.com/faizauthar12/paystore/config"
	"github.com/redis/go-redis/v9"
)

var createOrganizationQuery = `
	INSERT INTO organization (uuid, randid, created_at, updated_at, name, slug, fees_constant, fees_type)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
var updateOrganizationQuery = `UPDATE organization SET name = $1, slug = $2 WHERE uuid = $3`
var findOrganizationByUUIDQuery = `SELECT * FROM organization WHERE uuid = $1`
var findOrganizationBySlugQuery = `SELECT * FROM organization WHERE slug = $1`
var findOrganizationByNameQuery = `SELECT * FROM organization WHERE name = $1`

type RepositoryClient interface {
	Create(organization *Organization) error
	Update(organization *Organization) error
	FindByUUID(uuid string) (*Organization, error)
	FindBySlug(slug string) (*Organization, error)
	FindByName(name string) (*Organization, error)
}

type Repository struct {
	writeDB                    *sql.DB
	readDB                     *sql.DB
	base                       *redifu.Base[*Organization]
	createOrganizationStmt     *sql.Stmt
	updateOrganizationStmt     *sql.Stmt
	findOrganizationByUUIDStmt *sql.Stmt
	findOrganizationBySlugStmt *sql.Stmt
	findOrganizationByNameStmt *sql.Stmt
}

func (or *Repository) Create(organization *Organization) error {
	_, err := or.createOrganizationStmt.Exec(organization.GetUUID(),
		organization.GetRandId(), organization.GetCreatedAt(), organization.GetUUID(),
		organization.Name, organization.Slug, organization.FeesConstant, organization.FeesType)
	return err
}

func (or *Repository) Update(organization *Organization) error {
	_, err := or.updateOrganizationStmt.Exec(organization.Name, organization.Slug, organization.GetUUID())
	return err
}

func (or *Repository) FindByUUID(uuid string) (*Organization, error) {
	row, errScan := OrganizationRowScanner(or.findOrganizationByUUIDStmt.QueryRow(uuid))
	if errScan != nil {
		if errScan == sql.ErrNoRows {
			return nil, OrganizationNotFound
		}
		return nil, errScan
	}

	errSet := or.base.Set(row)
	if errSet != nil {
		return nil, errSet
	}
	return row, nil
}

func (or *Repository) FindBySlug(slug string) (*Organization, error) {
	row, errScan := OrganizationRowScanner(or.findOrganizationBySlugStmt.QueryRow(slug))
	if errScan != nil {
		if errScan == sql.ErrNoRows {
			return nil, OrganizationNotFound
		}
		
		return nil, errScan
	}

	errSet := or.base.Set(row)
	if errSet != nil {
		return nil, errSet
	}
	return row, nil
}

func (or *Repository) FindByName(name string) (*Organization, error) {
	row, errScan := OrganizationRowScanner(or.findOrganizationByNameStmt.QueryRow(name))
	if errScan != nil {
		return nil, errScan
	}

	errSet := or.base.Set(row)
	if errSet != nil {
		return nil, errSet
	}
	return row, nil
}

func OrganizationRowScanner(row *sql.Row) (*Organization, error) {
	org := NewOrganization()
	err := row.Scan(&org.UUID,
		&org.RandId, &org.CreatedAt, &org.UpdatedAt, &org.Name, &org.Slug, &org.FeesConstant, &org.FeesType)
	if err != nil {
		return nil, err
	}

	return org, nil
}

func NewRepository(writeDB *sql.DB, readDB *sql.DB, redis redis.UniversalClient, config *config.App) *Repository {
	base := redifu.NewBase[*Organization](redis, "organization:%s", config.RecordAge)

	createOrganizationStmt, err := writeDB.Prepare(createOrganizationQuery)
	if err != nil {
		panic(err)
	}
	updateOrganizationStmt, err := writeDB.Prepare(updateOrganizationQuery)
	if err != nil {
		panic(err)
	}
	findOrganizationByUUIDStmt, err := readDB.Prepare(findOrganizationByUUIDQuery)
	if err != nil {
		panic(err)
	}
	findOrganizationBySlugStmt, err := readDB.Prepare(findOrganizationBySlugQuery)
	if err != nil {
		panic(err)
	}

	organizationRepo := &Repository{
		writeDB:                    writeDB,
		readDB:                     readDB,
		base:                       base,
		createOrganizationStmt:     createOrganizationStmt,
		updateOrganizationStmt:     updateOrganizationStmt,
		findOrganizationByUUIDStmt: findOrganizationByUUIDStmt,
		findOrganizationBySlugStmt: findOrganizationBySlugStmt,
	}

	return organizationRepo
}
