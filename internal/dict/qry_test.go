package dict

import "testing"

func TestBuildSQLSimple(t *testing.T) {
	d := QryDef{
		Code: "cp_ima",
		Cols: []QryCol{
			{No: 1, Table: "ima_file", Field: "ima01", Width: 40, Return: true},
			{No: 2, Table: "ima_file", Field: "ima02", Width: 120},
			{No: 3, Table: "ima_file", Field: "ima021", Width: 120},
		},
	}
	got := d.BuildSQL(QrySQLOpt{})
	want := "SELECT 'N',ima01,ima02,ima021 FROM ima_file WHERE 1=1 ORDER BY ima01"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestBuildSQLJoinDistinctAndWhere(t *testing.T) {
	d := QryDef{
		Code:       "q_boh03",
		InputWhere: "bma01=bmb01 and bma10 in('1','2')\n and bma01=ima01 and ima151='Y'",
		JoinTables: "bma_file,bmb_file",
		Distinct:   true,
		Cols: []QryCol{
			{No: 1, Table: "ima_file", Field: "ima01", Return: true},
			{No: 2, Table: "ima_file", Field: "ima02"},
		},
	}
	got := d.BuildSQL(QrySQLOpt{})
	want := "SELECT UNIQUE 'N',ima01,ima02 FROM ima_file,bma_file,bmb_file WHERE " +
		"( bma01=bmb01 and bma10 in('1','2') and bma01=ima01 and ima151='Y' ) ORDER BY ima01"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestBuildSQLArgsAndConstruct(t *testing.T) {
	d := QryDef{
		Code:        "q_tqt02",
		InputWhere:  "tqa03='3'AND tqaacti='Y' AND tqa01 =tqh02 AND tqh01 ='arg1'",
		JoinTables:  "tqh_file",
		Distinct:    true,
		ConstrWhere: "tqa03='3'",
		Cols: []QryCol{
			{No: 1, Table: "tqa_file", Field: "tqa01", Return: true},
			{No: 2, Table: "tqa_file", Field: "tqa02"},
		},
	}
	var args [9]string
	args[0] = "TQ001"

	got := d.BuildSQL(QrySQLOpt{Args: args})
	want := "SELECT UNIQUE 'N',tqa01,tqa02 FROM tqa_file,tqh_file WHERE " +
		"( tqa03='3'AND tqaacti='Y' AND tqa01 =tqh02 AND tqh01 ='TQ001' ) ORDER BY tqa01"
	if got != want {
		t.Fatalf("args got  %q\nwant %q", got, want)
	}

	// 开窗（CONSTRUCT）模式换用 gab05，并叠加调用方条件
	got = d.BuildSQL(QrySQLOpt{Construct: true, Where: "tqa02 like 'A%'"})
	want = "SELECT UNIQUE 'N',tqa01,tqa02 FROM tqa_file,tqh_file WHERE " +
		"( tqa02 like 'A%' ) AND ( tqa03='3' ) ORDER BY tqa01"
	if got != want {
		t.Fatalf("construct got  %q\nwant %q", got, want)
	}
}

func TestBuildSQLSchemaOuterAndDate(t *testing.T) {
	d := QryDef{
		Code:       "q_demo",
		InputWhere: "occ01=ima01 and imaacti='Y'",
		Cols: []QryCol{
			{No: 1, Table: "ima_file", Field: "ima01", Return: true},
			{No: 2, Table: "ima_file", Field: "ima06", Type: 2, DateType: true},
			{No: 3, Table: "occ_file", Field: "occ02", Outer: true},
		},
	}
	got := d.BuildSQL(QrySQLOpt{Schema: "cof_demo", Oracle: true})
	want := "SELECT 'N',ima01,to_char(ima06,'yy/mm/dd'),occ02 FROM cof_demo.ima_file" +
		",cof_demo.occ_file WHERE " +
		"( cof_demo.occ_file.occ01(+)=ima01 and imaacti='Y' ) ORDER BY ima01"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}
