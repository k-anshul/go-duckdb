package duckdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testTypeValues struct {
	input  string
	output string
}

type testTypeInfo struct {
	TypeInfo
	testTypeValues
}

var testPrimitiveSQLValues = map[Type]testTypeValues{
	TYPE_BOOLEAN:      {input: `true::BOOLEAN`, output: `true`},
	TYPE_TINYINT:      {input: `42::TINYINT`, output: `42`},
	TYPE_SMALLINT:     {input: `42::SMALLINT`, output: `42`},
	TYPE_INTEGER:      {input: `42::INTEGER`, output: `42`},
	TYPE_BIGINT:       {input: `42::BIGINT`, output: `42`},
	TYPE_UTINYINT:     {input: `43::UTINYINT`, output: `43`},
	TYPE_USMALLINT:    {input: `43::USMALLINT`, output: `43`},
	TYPE_UINTEGER:     {input: `43::UINTEGER`, output: `43`},
	TYPE_UBIGINT:      {input: `43::UBIGINT`, output: `43`},
	TYPE_FLOAT:        {input: `1.7::FLOAT`, output: `1.7`},
	TYPE_DOUBLE:       {input: `1.7::DOUBLE`, output: `1.7`},
	TYPE_TIMESTAMP:    {input: `TIMESTAMP '1992-09-20 11:30:00.123456789'`, output: `1992-09-20 11:30:00.123456`},
	TYPE_DATE:         {input: `DATE '1992-09-20 11:30:00.123456789'`, output: `1992-09-20`},
	TYPE_TIME:         {input: `TIME '1992-09-20 11:30:00.123456789'`, output: `11:30:00.123456`},
	TYPE_INTERVAL:     {input: `INTERVAL 1 YEAR`, output: `1 year`},
	TYPE_HUGEINT:      {input: `44::HUGEINT`, output: `44`},
	TYPE_VARCHAR:      {input: `'hello world'::VARCHAR`, output: `hello world`},
	TYPE_BLOB:         {input: `'\xAA'::BLOB`, output: `\xAA`},
	TYPE_TIMESTAMP_S:  {input: `TIMESTAMP_S '1992-09-20 11:30:00.123456789'`, output: `1992-09-20 11:30:00`},
	TYPE_TIMESTAMP_MS: {input: `TIMESTAMP_MS '1992-09-20 11:30:00.123456789'`, output: `1992-09-20 11:30:00.123`},
	TYPE_TIMESTAMP_NS: {input: `TIMESTAMP_NS '1992-09-20 11:30:00.123456789'`, output: `1992-09-20 11:30:00.123456789`},
	TYPE_UUID:         {input: `uuid()`, output: ``},
	TYPE_TIMESTAMP_TZ: {input: `TIMESTAMPTZ '1992-09-20 11:30:00.123456789'`, output: `1992-09-20 11:30:00.123456+00`},
}

func getTypeInfos(t *testing.T) []testTypeInfo {
	var primitiveTypes []Type
	for k := range typeToStringMap {
		_, inMap := unsupportedTypeToStringMap[k]
		if inMap {
			continue
		}
		switch k {
		case TYPE_DECIMAL, TYPE_ENUM, TYPE_LIST, TYPE_STRUCT, TYPE_MAP:
			continue
		}
		primitiveTypes = append(primitiveTypes, k)
	}

	// Create each primitive type information.
	var typeInfos []testTypeInfo
	for _, primitive := range primitiveTypes {
		typeInfo, err := PrimitiveTypeInfo(primitive)
		require.NoError(t, err)
		info := testTypeInfo{
			TypeInfo:       typeInfo,
			testTypeValues: testPrimitiveSQLValues[typeInfo.t],
		}
		typeInfos = append(typeInfos, info)
	}

	// Create nested types.
	decimalInfo := testTypeInfo{
		TypeInfo: DecimalTypeInfo(3, 2),
		testTypeValues: testTypeValues{
			input:  `4::DECIMAL(3, 2)`,
			output: `4.00`,
		},
	}

	names := []string{"hello", "world"}
	info, err := EnumTypeInfo(names)
	enumInfo := testTypeInfo{
		TypeInfo: info,
		testTypeValues: testTypeValues{
			input:  `'hello'::greeting`,
			output: `hello`,
		},
	}
	require.NoError(t, err)

	info, err = ListTypeInfo(decimalInfo.TypeInfo)
	listInfo := testTypeInfo{
		TypeInfo: info,
		testTypeValues: testTypeValues{
			input:  `[4::DECIMAL(3, 2)]`,
			output: `[4.00]`,
		},
	}
	require.NoError(t, err)

	info, err = ListTypeInfo(listInfo.TypeInfo)
	nestedListInfo := testTypeInfo{
		TypeInfo: info,
		testTypeValues: testTypeValues{
			input:  `[[4::DECIMAL(3, 2)]]`,
			output: `[[4.00]]`,
		},
	}
	require.NoError(t, err)

	childTypeInfos := []TypeInfo{enumInfo.TypeInfo, nestedListInfo.TypeInfo}
	info, err = StructTypeInfo(childTypeInfos, names)
	structTypeInfo := testTypeInfo{
		TypeInfo: info,
		testTypeValues: testTypeValues{
			input:  `{'hello': 'hello'::greeting, 'world': [[4::DECIMAL(3, 2)]]}`,
			output: `{'hello': hello, 'world': [[4.00]]}`,
		},
	}
	require.NoError(t, err)

	nestedChildTypeInfos := []TypeInfo{structTypeInfo.TypeInfo, listInfo.TypeInfo}
	info, err = StructTypeInfo(nestedChildTypeInfos, names)
	nestedStructTypeInfo := testTypeInfo{
		TypeInfo: info,
		testTypeValues: testTypeValues{
			input: `{
						'hello': {'hello': 'hello'::greeting, 'world': [[4::DECIMAL(3, 2)]]},
						'world': [4::DECIMAL(3, 2)]
					}`,
			output: `{'hello': {'hello': hello, 'world': [[4.00]]}, 'world': [4.00]}`,
		},
	}
	require.NoError(t, err)

	info, err = MapTypeInfo(decimalInfo.TypeInfo, nestedStructTypeInfo.TypeInfo)
	mapTypeInfo := testTypeInfo{
		TypeInfo: info,
		testTypeValues: testTypeValues{
			input: `MAP {
					4::DECIMAL(3, 2) : {
						'hello': {'hello': 'hello'::greeting, 'world': [[4::DECIMAL(3, 2)]]},
						'world': [4::DECIMAL(3, 2)]
					}
					}`,
			output: `{4.00={'hello': {'hello': hello, 'world': [[4.00]]}, 'world': [4.00]}}`,
		},
	}
	require.NoError(t, err)

	typeInfos = append(typeInfos, decimalInfo, enumInfo, listInfo, nestedListInfo, structTypeInfo, nestedStructTypeInfo, mapTypeInfo)
	return typeInfos
}

func TestTypeInterface(t *testing.T) {
	typeInfos := getTypeInfos(t)

	// Use each type as a child.
	for _, info := range typeInfos {
		_, err := ListTypeInfo(info.TypeInfo)
		require.NoError(t, err)
	}
}
