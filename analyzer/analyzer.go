package analyzer

import (
	"errors"
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"

	"github.com/aslakhellesoy/mermerd/config"
	"github.com/aslakhellesoy/mermerd/database"
	"github.com/aslakhellesoy/mermerd/presentation"
	"github.com/aslakhellesoy/mermerd/util"
)

type analyzer struct {
	loadingSpinner   presentation.LoadingSpinner
	config           config.MermerdConfig
	connectorFactory database.ConnectorFactory
	questioner       Questioner
}

type Analyzer interface {
	Analyze() (*database.Result, error)
	GetConnectionString() (string, error)
	GetSchemas(db database.Connector) ([]string, error)
	GetTables(db database.Connector, selectedSchemas []string) ([]database.TableDetail, error)
	GetColumnsAndConstraints(db database.Connector, selectedTables []database.TableDetail) ([]database.TableResult, error)
}

func NewAnalyzer(config config.MermerdConfig, connectorFactory database.ConnectorFactory, questioner Questioner) Analyzer {
	loadingSpinner := presentation.NewLoadingSpinner()
	return analyzer{loadingSpinner, config, connectorFactory, questioner}
}

func (a analyzer) Analyze() (*database.Result, error) {
	connectionString, err := a.GetConnectionString()
	if err != nil {
		return nil, err
	}

	db, err := a.connectorFactory.NewConnector(connectionString)
	if err != nil {
		return nil, err
	}

	a.loadingSpinner.Start("Connecting to database")
	if err = db.Connect(); err != nil {
		return nil, err
	}
	defer db.Close()
	a.loadingSpinner.Stop()

	selectedSchemas, err := a.GetSchemas(db)
	if err != nil {
		return nil, err
	}

	selectedTables, err := a.GetTables(db, selectedSchemas)
	if err != nil {
		return nil, err
	}
	// sort the tables so the output is more deterministic
	sortTables(selectedTables)

	tableResults, err := a.GetColumnsAndConstraints(db, selectedTables)
	if err != nil {
		return nil, err
	}

	return &database.Result{Tables: tableResults}, nil
}

func sortTables(tables []database.TableDetail) {
	sort.SliceStable(tables, func(i, j int) bool {
		if tables[i].Schema != tables[j].Schema {
			return tables[i].Schema < tables[j].Schema
		}
		return tables[i].Name < tables[j].Name
	})

}

func (a analyzer) GetConnectionString() (string, error) {
	if connectionString := a.config.ConnectionString(); connectionString != "" {
		return connectionString, nil
	}

	return a.questioner.AskConnectionQuestion(a.config.ConnectionStringSuggestions())
}

func (a analyzer) GetSchemas(db database.Connector) ([]string, error) {
	if selectedSchema := a.config.Schemas(); len(selectedSchema) > 0 {
		return selectedSchema, nil
	}

	a.loadingSpinner.Start("Getting schemas")
	schemas, err := db.GetSchemas()
	a.loadingSpinner.Stop()
	if err != nil {
		logrus.Error("Getting schemas failed", " | ", err)
		return []string{}, err
	}

	logrus.WithField("count", len(schemas)).Info("Got schemas")
	if a.config.UseAllSchemas() {
		return schemas, nil
	}

	switch len(schemas) {
	case 0:
		return []string{}, errors.New("no schemas available")
	case 1:
		return schemas, nil
	default:
		return a.questioner.AskSchemaQuestion(schemas)
	}
}

func (a analyzer) GetTables(db database.Connector, selectedSchemas []string) ([]database.TableDetail, error) {
	if selectedTables := a.config.SelectedTables(); len(selectedTables) > 0 {
		return util.Map2(selectedTables, func(value string) database.TableDetail {
			res, err := database.ParseTableName(value, selectedSchemas)
			if err != nil {
				logrus.Error("Could not parse table name", value)
			}

			return res
		}), nil
	}

	a.loadingSpinner.Start("Getting tables")
	tables, err := db.GetTables(selectedSchemas)
	a.loadingSpinner.Stop()
	if err != nil {
		logrus.Error("Getting tables failed", " | ", err)
		return nil, err
	}

	if len(tables) == 0 {
		logrus.Error("No tables found")
	}

	logrus.WithField("count", len(tables)).Info("Got tables")

	if a.config.UseAllTables() {
		return tables, nil
	}

	tableNames := util.Map2(tables, func(table database.TableDetail) string {
		return fmt.Sprintf("%s.%s", table.Schema, table.Name)
	})
	surveyResult, err := a.questioner.AskTableQuestion(tableNames)
	if err != nil {
		return []database.TableDetail{}, err
	}
	return util.Map2(surveyResult, func(value string) database.TableDetail {
		res, err := database.ParseTableName(value, selectedSchemas)
		if err != nil {
			logrus.Error("Could not parse table name", value)
		}

		return res
	}), nil
}

func (a analyzer) GetColumnsAndConstraints(db database.Connector, selectedTables []database.TableDetail) ([]database.TableResult, error) {
	var tableResults []database.TableResult
	a.loadingSpinner.Start("Getting columns and constraints")
	for _, table := range selectedTables {
		columns, err := db.GetColumns(table)
		if err != nil {
			logrus.Error("Getting columns failed", " | ", err)
			return nil, err
		}

		constraints, err := db.GetConstraints(table)
		if err != nil {
			logrus.Error("Getting constraints failed", " | ", err)
			return nil, err
		}

		sortColumns(columns)
		tableResults = append(tableResults, database.TableResult{Table: table, Columns: columns, Constraints: constraints})
	}
	a.loadingSpinner.Stop()
	columnCount, constraintCount := getTableResultStats(tableResults)
	logrus.WithFields(logrus.Fields{"columns": columnCount, "constraints": constraintCount}).Info("Got columns and constraints")
	return tableResults, nil
}

func getTableResultStats(tableResults []database.TableResult) (columnCount int, constraintCount int) {
	for _, tableResult := range tableResults {
		columnCount += len(tableResult.Columns)
		constraintCount += len(tableResult.Constraints)
	}

	return columnCount, constraintCount
}

func sortColumns(columns []database.ColumnResult) {
	sort.SliceStable(columns, func(i, j int) bool {
		return columns[i].Name < columns[j].Name
	})
}
