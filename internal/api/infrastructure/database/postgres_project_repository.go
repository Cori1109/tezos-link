package database

import (
	"database/sql"
	"fmt"
	"github.com/octo-technology/tezos-link/backend/internal/api/domain/model"
	"github.com/octo-technology/tezos-link/backend/internal/api/domain/repository"
	"github.com/octo-technology/tezos-link/backend/pkg/domain/errors"
	"github.com/sirupsen/logrus"
)

type postgresProjectRepository struct {
	connection *sql.DB
}

// NewPostgresProjectRepository returns a new postgres project repository
func NewPostgresProjectRepository(connection *sql.DB) repository.ProjectRepository {
	return &postgresProjectRepository{
		connection: connection,
	}
}

// FindAll returns all projects
func (pg *postgresProjectRepository) FindAll() ([]*model.Project, error) {
	rows, err := pg.connection.Query("SELECT id, name, uuid FROM projects")
	if err != nil {
		return nil, fmt.Errorf("no projects found: %s", err)
	}

	var r []*model.Project
	for rows.Next() {
		cur := model.Project{}
		err := rows.Scan(&cur.ID, &cur.Name, &cur.UUID)
		if err != nil {
			return nil, fmt.Errorf("could not map projects: %s", err)
		}
		r = append(r, &cur)
	}

	return r, nil
}

// FindByUUID finds a project by uuid
func (pg *postgresProjectRepository) FindByUUID(uuid string) (*model.Project, error) {
	r := model.Project{}
	err := pg.connection.
		QueryRow("SELECT id, name, uuid FROM projects WHERE uuid = $1", uuid).
		Scan(&r.ID, &r.Name, &r.UUID)

	if err != nil {
		logrus.Errorf("project %s not found: %s", uuid, err)
		return nil, errors.ErrProjectNotFound
	}

	return &r, nil
}

// Save insert a new project
func (pg *postgresProjectRepository) Save(name string, uuid string) (*model.Project, error) {
	r := model.Project{}

	err := pg.connection.
		QueryRow("INSERT INTO projects(name, uuid) VALUES ($1, $2) RETURNING id, name, uuid", name, uuid).
		Scan(&r.ID, &r.Name, &r.UUID)

	if err != nil {
		return nil, fmt.Errorf("could not insert project %s: %s", name, err)
	}

	return &r, nil
}

// UpdateByID update a project by id
func (pg *postgresProjectRepository) UpdateByID(project *model.Project) error {
	panic("implement me")
}

// DeleteByID delete a project by id
func (pg *postgresProjectRepository) DeleteByID(project *model.Project) error {
	panic("implement me")
}

// Ping ping the database
func (pg *postgresProjectRepository) Ping() error {
	err := pg.connection.Ping()
	if err != nil {
		return err
	}

	return nil
}
