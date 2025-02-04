package diagram

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aslakhellesoy/mermerd/database"
	"github.com/aslakhellesoy/mermerd/mocks"
)

func TestGetRelation(t *testing.T) {
	testCases := []struct {
		isPrimary        bool
		hasMultiplePK    bool
		expectedRelation ErdRelationType
	}{
		{true, true, relationManyToOne},
		{false, true, relationManyToOne},
		{false, false, relationManyToOne},
		{true, false, relationOneToOne},
	}

	for index, testCase := range testCases {
		t.Run(fmt.Sprintf("run #%d", index), func(t *testing.T) {
			// Arrange
			constraint := database.ConstraintResult{
				FkTable:        "tableA",
				PkTable:        "tableB",
				ConstraintName: "constraintXY",
				IsPrimary:      testCase.isPrimary,
				HasMultiplePK:  testCase.hasMultiplePK,
			}

			// Act
			result := getRelation(constraint)

			// Assert
			assert.Equal(t, testCase.expectedRelation, result)
		})
	}
}

func TestGetAttributeKey(t *testing.T) {
	testCases := []struct {
		column                  database.ColumnResult
		expectedAttributeResult ErdAttributeKey
	}{
		{
			column: database.ColumnResult{
				Name:      "",
				DataType:  "",
				IsPrimary: true,
				IsForeign: false,
			},
			expectedAttributeResult: primaryKey,
		},
		{
			column: database.ColumnResult{
				Name:      "",
				DataType:  "",
				IsPrimary: false,
				IsForeign: true,
			},
			expectedAttributeResult: foreignKey,
		},
		{
			column: database.ColumnResult{
				Name:      "",
				DataType:  "",
				IsPrimary: true,
				IsForeign: true,
			},
			expectedAttributeResult: primaryKey,
		},
		{
			column: database.ColumnResult{
				Name:      "",
				DataType:  "",
				IsPrimary: false,
				IsForeign: false,
			},
			expectedAttributeResult: none,
		},
	}

	for index, testCase := range testCases {
		t.Run(fmt.Sprintf("run #%d", index), func(t *testing.T) {
			// Arrange
			column := testCase.column

			// Act
			result := getAttributeKey(column)

			// Assert
			assert.Equal(t, testCase.expectedAttributeResult, result)
		})
	}
}

func TestTableNameInSlice(t *testing.T) {
	t.Run("Existing item should be found", func(t *testing.T) {
		// Arrange
		tableName := "testTable"
		slice := []ErdTableData{{Name: tableName}}

		// Act
		result := tableNameInSlice(slice, tableName)

		// Assert
		assert.True(t, result)
	})

	t.Run("Missing item should not be found", func(t *testing.T) {
		// Arrange
		tableName := "testTable"
		slice := []ErdTableData{{Name: "notTheTableName"}}

		// Act
		result := tableNameInSlice(slice, tableName)

		// Assert
		assert.False(t, result)
	})
}

func TestGetColumnData(t *testing.T) {
	columnName := "testColumn"
	enumValues := "a,b"
	comment := `{"comment":"detail"}`
	expectedComment := "{#quot;comment#quot;:#quot;detail#quot;}"

	column := database.ColumnResult{
		Name:       columnName,
		IsPrimary:  true,
		EnumValues: enumValues,
		Comment:    comment,
	}

	t.Run("Get all fields", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("OmitAttributeKeys").Return(false).Once()
		configMock.On("ShowDescriptions").Return([]string{"enumValues", "columnComments"}).Once()

		// Act
		result := getColumnData(&configMock, column)

		// Assert
		configMock.AssertExpectations(t)
		assert.Equal(t, columnName, result.Name)
		assert.Equal(t, "<"+enumValues+"> "+expectedComment, result.Description)
		assert.Equal(t, primaryKey, result.AttributeKey)
	})

	t.Run("Get all fields with enum values", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("OmitAttributeKeys").Return(false).Once()
		configMock.On("ShowDescriptions").Return([]string{"enumValues"}).Once()

		// Act
		result := getColumnData(&configMock, column)

		// Assert
		configMock.AssertExpectations(t)
		assert.Equal(t, columnName, result.Name)
		assert.Equal(t, "<"+enumValues+">", result.Description)
		assert.Equal(t, primaryKey, result.AttributeKey)
	})

	t.Run("Get all fields with column comments", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("OmitAttributeKeys").Return(false).Once()
		configMock.On("ShowDescriptions").Return([]string{"columnComments"}).Once()

		// Act
		result := getColumnData(&configMock, column)

		// Assert
		configMock.AssertExpectations(t)
		assert.Equal(t, columnName, result.Name)
		assert.Equal(t, expectedComment, result.Description)
		assert.Equal(t, primaryKey, result.AttributeKey)
	})

	t.Run("Get all fields except description", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("OmitAttributeKeys").Return(false).Once()
		configMock.On("ShowDescriptions").Return([]string{""}).Once()

		// Act
		result := getColumnData(&configMock, column)

		// Assert
		configMock.AssertExpectations(t)
		assert.Equal(t, columnName, result.Name)
		assert.Equal(t, "", result.Description)
		assert.Equal(t, primaryKey, result.AttributeKey)
	})

	t.Run("Get all fields except attribute key", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("OmitAttributeKeys").Return(true).Once()
		configMock.On("ShowDescriptions").Return([]string{"enumValues", "columnComments"}).Once()

		// Act
		result := getColumnData(&configMock, column)

		// Assert
		configMock.AssertExpectations(t)
		assert.Equal(t, columnName, result.Name)
		assert.Equal(t, "<"+enumValues+"> "+expectedComment, result.Description)
		assert.Equal(t, none, result.AttributeKey)
	})

	t.Run("Get only minimal fields", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("OmitAttributeKeys").Return(true).Once()
		configMock.On("ShowDescriptions").Return([]string{""}).Once()

		// Act
		result := getColumnData(&configMock, column)

		// Assert
		configMock.AssertExpectations(t)
		assert.Equal(t, columnName, result.Name)
		assert.Equal(t, "", result.Description)
		assert.Equal(t, none, result.AttributeKey)
	})
}

func TestShouldSkipConstraint(t *testing.T) {
	tableName1 := "Table1"
	tableName2 := "Table2"
	tables := []ErdTableData{{Name: tableName1}, {Name: tableName2}}

	t.Run("ShowAllConstraints config should never skip", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("ShowAllConstraints").Return(true).Once()
		constraint := database.ConstraintResult{PkTable: tableName1}

		// Act
		result := shouldSkipConstraint(&configMock, tables, constraint)

		// Assert
		configMock.AssertExpectations(t)
		assert.False(t, result)
	})

	t.Run("Skip constraint if both tables are not present", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("ShowAllConstraints").Return(false).Once()
		constraint := database.ConstraintResult{PkTable: tableName1, FkTable: "UnknownTable"}

		// Act
		result := shouldSkipConstraint(&configMock, tables, constraint)

		// Assert
		configMock.AssertExpectations(t)
		assert.True(t, result)
	})

	t.Run("Do not skip constraint if both tables are present", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("ShowAllConstraints").Return(false).Once()
		constraint := database.ConstraintResult{PkTable: tableName1, FkTable: tableName2}

		// Act
		result := shouldSkipConstraint(&configMock, tables, constraint)

		// Assert
		configMock.AssertExpectations(t)
		assert.False(t, result)
	})
}

func TestGetConstraintData(t *testing.T) {
	t.Run("OmitConstraintLabels should remove the constraint label", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("OmitConstraintLabels").Return(true).Once()
		configMock.On("ShowSchemaPrefix").Return(false).Twice()
		constraint := database.ConstraintResult{ColumnName: "Column1"}

		// Act
		result := getConstraintData(&configMock, constraint)

		// Assert
		configMock.AssertExpectations(t)
		assert.Equal(t, result.ConstraintLabel, "")
	})
}

func TestGetTableName(t *testing.T) {
	t.Run("Do not show schema prefix if config not active", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("ShowSchemaPrefix").Return(false).Once()
		tableDetail := database.TableDetail{Schema: "SchemaName", Name: "TableName"}

		// Act
		result := getTableName(&configMock, tableDetail)

		// Assert
		configMock.AssertExpectations(t)
		assert.Equal(t, "TableName", result)
	})

	t.Run("Show schema prefix if config is active", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("ShowSchemaPrefix").Return(true).Once()
		configMock.On("SchemaPrefixSeparator").Return("_").Once()
		tableDetail := database.TableDetail{Schema: "SchemaName", Name: "TableName"}

		// Act
		result := getTableName(&configMock, tableDetail)

		// Assert
		configMock.AssertExpectations(t)
		assert.Equal(t, "SchemaName_TableName", result)
	})

	t.Run("Show escaped schema prefix if config is active and separator is a full stop", func(t *testing.T) {
		// Arrange
		configMock := mocks.MermerdConfig{}
		configMock.On("ShowSchemaPrefix").Return(true).Once()
		configMock.On("SchemaPrefixSeparator").Return(".").Once()
		tableDetail := database.TableDetail{Schema: "SchemaName", Name: "TableName"}

		// Act
		result := getTableName(&configMock, tableDetail)

		// Assert
		configMock.AssertExpectations(t)
		assert.Equal(t, "\"SchemaName.TableName\"", result)
	})

}
