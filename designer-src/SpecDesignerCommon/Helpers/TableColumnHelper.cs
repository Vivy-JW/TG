using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text.RegularExpressions;
using System.Xml.Linq;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x02000145 RID: 325
	public class TableColumnHelper
	{
		// Token: 0x06000B56 RID: 2902 RVA: 0x00037C08 File Offset: 0x00035E08
		public static void Parse(string fileName, string settingPath)
		{
			TableColumnHelper.tables.Clear();
			TableColumnHelper.columns.Clear();
			TableColumnHelper.BasePath = settingPath;
			XElement xelement = XElement.Load(fileName);
			TableColumnHelper.parseTablesInfo(xelement);
		}

		// Token: 0x06000B57 RID: 2903 RVA: 0x00037C3C File Offset: 0x00035E3C
		private static void parseTablesInfo(XElement tablesInfo)
		{
			TableColumnHelper.tables.Clear();
			IEnumerable<XElement> enumerable = tablesInfo.Elements("table");
			foreach (XElement xelement in enumerable)
			{
				string value = xelement.Attribute("name").Value;
				TableColumnHelper.tables.Add(value, xelement);
			}
		}

		// Token: 0x06000B58 RID: 2904 RVA: 0x00037CBC File Offset: 0x00035EBC
		public static XElement FindTableColumns(string tableName)
		{
			XElement xelement = null;
			XElement xelement2 = null;
			if (!TableColumnHelper.tables.TryGetValue(tableName, out xelement))
			{
				return null;
			}
			if (TableColumnHelper.columns.TryGetValue(tableName, out xelement2))
			{
				return xelement2;
			}
			string value = xelement.Attribute("module").Value;
			string text = Path.Combine(TableColumnHelper.BasePath, value, "tbl", tableName + ".tbl");
			if (File.Exists(text))
			{
				using (StreamReader streamReader = new StreamReader(text))
				{
					xelement2 = XElement.Parse(streamReader.ReadToEnd());
					TableColumnHelper.columns.Add(tableName, xelement2);
					streamReader.Close();
				}
			}
			return xelement2;
		}

		// Token: 0x06000B59 RID: 2905 RVA: 0x00037D9C File Offset: 0x00035F9C
		public static IEnumerable<XElement> FindTableColumns(string tableName, string env)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(tableName);
			if (xelement == null)
			{
				return null;
			}
			if (env.Equals("s"))
			{
				return xelement.Elements("column").ToArray<XElement>();
			}
			Regex regex = new Regex("^[a-z]{4}ua[0-9]{3}$");
			Regex regex2 = new Regex("^[a-z]{4}ud[0-9]{3}$");
			Dictionary<string, int> columnOrderDict = new Dictionary<string, int>();
			foreach (XElement xelement2 in xelement.Elements("column"))
			{
				string value = xelement2.Attribute("name").Value;
				if (regex2.IsMatch(value))
				{
					columnOrderDict.Add(value, 1);
				}
				else if (regex.IsMatch(value))
				{
					columnOrderDict.Add(value, 2);
				}
				else
				{
					columnOrderDict.Add(value, 3);
				}
			}
			return from x in xelement.Elements("column")
				orderby columnOrderDict[x.Attribute("name").Value]
				select x;
		}

		// Token: 0x06000B5A RID: 2906 RVA: 0x00037EC0 File Offset: 0x000360C0
		public static IEnumerable<XElement> GetTables()
		{
			if (TableColumnHelper.tables.Count == 0)
			{
				return null;
			}
			return TableColumnHelper.tables.Values.ToArray<XElement>();
		}

		// Token: 0x06000B5B RID: 2907 RVA: 0x00037F0C File Offset: 0x0003610C
		public static IEnumerable<XElement> GetOrderedTables(SpecificationInfo specInfo)
		{
			if (TableColumnHelper.tables.Count == 0)
			{
				return null;
			}
			IEnumerable<XElement> enumerable = TableColumnHelper.tables.Values.ToArray<XElement>();
			if (specInfo == null)
			{
				return enumerable;
			}
			TableAssociationModel associateTable = specInfo.AssociateTable;
			Dictionary<string, int> tableOrderDict = new Dictionary<string, int>();
			foreach (string text in TableColumnHelper.tables.Keys)
			{
				if (text.Equals("type_t"))
				{
					tableOrderDict.Add(text, 2);
				}
				else
				{
					tableOrderDict.Add(text, 99);
				}
			}
			foreach (TBLModel tblmodel in associateTable.AliveTBLs)
			{
				XElement xelement;
				if (TableColumnHelper.tables.TryGetValue(tblmodel.TBLName, out xelement))
				{
					tableOrderDict[tblmodel.TBLName] = 1;
				}
			}
			return TableColumnHelper.tables.Values.OrderBy<XElement, int>((XElement x) => tableOrderDict[x.Attribute("name").Value]).ToArray<XElement>();
		}

		// Token: 0x06000B5C RID: 2908 RVA: 0x00038050 File Offset: 0x00036250
		public static IEnumerable<XElement> GetColFields(string table)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement == null || xelement.Element("col_attr") == null)
			{
				return null;
			}
			return from field in xelement.Element("col_attr").Elements("field")
				select (field);
		}

		// Token: 0x06000B5D RID: 2909 RVA: 0x000380C4 File Offset: 0x000362C4
		public static IEnumerable<XElement> GetRefFields(string table)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement == null || xelement.Element("ref_field") == null)
			{
				return null;
			}
			return from field in xelement.Element("ref_field").Elements("field")
				select (field);
		}

		// Token: 0x06000B5E RID: 2910 RVA: 0x00038138 File Offset: 0x00036338
		public static IEnumerable<XElement> GetLangFields(string table)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement == null || xelement.Element("multi_lang") == null)
			{
				return null;
			}
			return from field in xelement.Element("multi_lang").Elements("field")
				select (field);
		}

		// Token: 0x06000B5F RID: 2911 RVA: 0x000381AC File Offset: 0x000363AC
		public static IEnumerable<XElement> GetSCCFields(string table)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement == null || xelement.Element("scc") == null)
			{
				return null;
			}
			return from field in xelement.Element("scc").Elements("field").Elements<XElement>("code")
				select (field);
		}

		// Token: 0x06000B60 RID: 2912 RVA: 0x0003822C File Offset: 0x0003642C
		public static XElement GetTreeNode(string table)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement == null || xelement.Element("tree") == null)
			{
				return null;
			}
			return xelement.Element("tree");
		}

		// Token: 0x06000B61 RID: 2913 RVA: 0x00038268 File Offset: 0x00036468
		public static string GetTableDesc(string tableName)
		{
			XElement xelement = null;
			if (TableColumnHelper.tables.Count == 0 || !TableColumnHelper.tables.TryGetValue(tableName, out xelement))
			{
				return null;
			}
			return xelement.Attribute("desc").Value;
		}

		// Token: 0x06000B62 RID: 2914 RVA: 0x000382D4 File Offset: 0x000364D4
		public static string GetColumnText(string tableName, string columnName)
		{
			if (columnName == null)
			{
				return string.Empty;
			}
			XElement xelement = null;
			if (!TableColumnHelper.columns.TryGetValue(tableName, out xelement))
			{
				return null;
			}
			xelement = (from c in xelement.Elements("column")
				where columnName.Equals(c.Attribute("name").Value)
				select c).FirstOrDefault<XElement>();
			if (xelement == null)
			{
				return null;
			}
			return xelement.Attribute("text").Value;
		}

		// Token: 0x06000B63 RID: 2915 RVA: 0x00038350 File Offset: 0x00036550
		public static string GetColumnTextByFullName(string fullname)
		{
			string[] array = fullname.Split(new char[] { '.' });
			if (array.Length == 2)
			{
				string text = array[0];
				string text2 = array[1];
				XElement xelement = TableColumnHelper.FindTableColumns(text);
				if (xelement != null)
				{
					return TableColumnHelper.GetColumnText(text, text2);
				}
			}
			return string.Empty;
		}

		// Token: 0x06000B64 RID: 2916 RVA: 0x000383C4 File Offset: 0x000365C4
		internal static XElement FindRefField(string table, string column)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement == null || xelement.Element("ref_field") == null)
			{
				return null;
			}
			return (from r in xelement.Element("ref_field").Elements("field")
				where column.Equals((string)r.Attribute("depend_field"))
				select r).FirstOrDefault<XElement>();
		}

		// Token: 0x06000B65 RID: 2917 RVA: 0x00038460 File Offset: 0x00036660
		internal static XElement FindMultiLangField(string table, string column)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement == null || xelement.Element("multi_lang") == null)
			{
				return null;
			}
			return (from r in xelement.Element("multi_lang").Elements("field")
				where column.Equals((string)r.Attribute("depend_field"))
				select r).FirstOrDefault<XElement>();
		}

		// Token: 0x06000B66 RID: 2918 RVA: 0x000384FC File Offset: 0x000366FC
		internal static XElement FindHelpCodeField(string table, string column)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement == null || xelement.Element("help_code") == null)
			{
				return null;
			}
			return (from r in xelement.Element("help_code").Elements("field")
				where column.Equals((string)r.Attribute("depend_field"))
				select r).FirstOrDefault<XElement>();
		}

		// Token: 0x06000B67 RID: 2919 RVA: 0x00038598 File Offset: 0x00036798
		public static string GetDefaultAttributeValue(string tableName, string columnName, string attrName)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(tableName);
			if (xelement == null || xelement.Element("col_attr") == null)
			{
				return null;
			}
			XElement xelement2 = (from c in xelement.Element("col_attr").Elements("field")
				where c.Attribute("name").Value == columnName
				select c).FirstOrDefault<XElement>();
			if (xelement2 == null)
			{
				return null;
			}
			if (xelement2.Attribute(attrName) == null)
			{
				return null;
			}
			return xelement2.Attribute(attrName).Value;
		}

		// Token: 0x06000B68 RID: 2920 RVA: 0x00038658 File Offset: 0x00036858
		public static XElement GetColumnAttrInfo(string table, string column)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement != null && xelement.Element("col_attr") != null)
			{
				return (from f in xelement.Element("col_attr").Elements("field")
					where column.Equals((string)f.Attribute("name"))
					select f).FirstOrDefault<XElement>();
			}
			return null;
		}

		// Token: 0x06000B69 RID: 2921 RVA: 0x0003870C File Offset: 0x0003690C
		public static XElement GetColumnInfo(string table, string column)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(table);
			if (xelement != null)
			{
				return (from c in xelement.Elements("column")
					where c.Attribute("name") != null && column.Equals(c.Attribute("name").Value)
					select c).FirstOrDefault<XElement>();
			}
			return null;
		}

		// Token: 0x06000B6A RID: 2922 RVA: 0x0003878C File Offset: 0x0003698C
		public static bool IsPK(string tableName, string columnName)
		{
			if (columnName == null || tableName == null)
			{
				return false;
			}
			XElement xelement = null;
			if (!TableColumnHelper.columns.TryGetValue(tableName, out xelement))
			{
				return false;
			}
			xelement = (from c in xelement.Elements("column")
				where columnName.Equals(c.Attribute("name").Value)
				select c).FirstOrDefault<XElement>();
			return xelement != null && xelement.Attribute("pk").Value == "Y";
		}

		// Token: 0x06000B6B RID: 2923 RVA: 0x00038850 File Offset: 0x00036A50
		public static XElement GetColField(string table, string column)
		{
			IEnumerable<XElement> colFields = TableColumnHelper.GetColFields(table);
			if (colFields != null)
			{
				return colFields.Where<XElement>((XElement c) => c.Attribute("name") != null && c.Attribute("name").Value == column).FirstOrDefault<XElement>();
			}
			return null;
		}

		// Token: 0x06000B6C RID: 2924 RVA: 0x00038894 File Offset: 0x00036A94
		public static string GetPK(string tblName)
		{
			XElement xelement = null;
			if (!TableColumnHelper.tables.TryGetValue(tblName, out xelement))
			{
				return "";
			}
			if (xelement.Attribute("pk") == null)
			{
				return "";
			}
			return xelement.Attribute("pk").Value;
		}

		// Token: 0x06000B6D RID: 2925 RVA: 0x00038924 File Offset: 0x00036B24
		public static XElement GetFK(string master, string tblName)
		{
			XElement xelement = null;
			if (TableColumnHelper.tables.TryGetValue(tblName, out xelement))
			{
				XElement xelement2 = (from f in xelement.Elements("fk")
					where f.Attribute("fk_table") != null && f.Attribute("fk_table").Value == master
					select f).FirstOrDefault<XElement>();
				if (xelement2 != null)
				{
					return new XElement(xelement2);
				}
			}
			return null;
		}

		// Token: 0x06000B6E RID: 2926 RVA: 0x00038988 File Offset: 0x00036B88
		public static bool CheckColumnExist(string columnName)
		{
			bool flag = false;
			foreach (XElement xelement in TableColumnHelper.GetTables())
			{
				if (flag)
				{
					break;
				}
				string value = xelement.Attribute("name").Value;
				IEnumerable<XElement> colFields = TableColumnHelper.GetColFields(value);
				if (colFields != null)
				{
					foreach (XElement xelement2 in colFields)
					{
						if (flag)
						{
							break;
						}
						if (xelement2.Attribute("name").Value == columnName)
						{
							flag = true;
						}
					}
				}
			}
			return flag;
		}

		// Token: 0x06000B6F RID: 2927 RVA: 0x00038A54 File Offset: 0x00036C54
		public static void RemoveTableOrColumn(string name)
		{
			if (!TableColumnHelper.tables.ContainsKey(name))
			{
				TableColumnHelper.tables.Remove(name);
			}
			if (TableColumnHelper.columns.ContainsKey(name))
			{
				TableColumnHelper.columns.Remove(name);
			}
		}

		// Token: 0x06000B70 RID: 2928 RVA: 0x00038B00 File Offset: 0x00036D00
		public static bool IsVarcharField(string tableName, string columnName)
		{
			XElement xelement = TableColumnHelper.FindTableColumns(tableName);
			if (xelement != null)
			{
				string text = (from col in xelement.Elements("column")
					where col.Attribute("name") != null && col.Attribute("name").Value == columnName && col.Attribute("type") != null
					select col.Attribute("type").Value).FirstOrDefault<string>();
				return text != null && text.ToString().StartsWith("varchar2");
			}
			return true;
		}

		// Token: 0x04000462 RID: 1122
		private static Dictionary<string, XElement> tables = new Dictionary<string, XElement>();

		// Token: 0x04000463 RID: 1123
		private static Dictionary<string, XElement> columns = new Dictionary<string, XElement>();

		// Token: 0x04000464 RID: 1124
		public static string BasePath;
	}
}
