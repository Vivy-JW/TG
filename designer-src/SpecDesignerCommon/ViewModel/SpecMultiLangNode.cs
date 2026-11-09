using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000F2 RID: 242
	public class SpecMultiLangNode : AbstractSpecNode, IDataErrorInfo
	{
		// Token: 0x06000809 RID: 2057 RVA: 0x00023976 File Offset: 0x00021B76
		public SpecMultiLangNode(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x0600080A RID: 2058 RVA: 0x00023980 File Offset: 0x00021B80
		public SpecMultiLangNode(PackageKey key, XElement source, string src)
			: base(key, source, src)
		{
		}

		// Token: 0x17000234 RID: 564
		// (get) Token: 0x0600080B RID: 2059 RVA: 0x0002398B File Offset: 0x00021B8B
		// (set) Token: 0x0600080C RID: 2060 RVA: 0x00023998 File Offset: 0x00021B98
		public string DependField
		{
			get
			{
				return base.GetAttribute("depend_field");
			}
			set
			{
				this.AttributeChanged("depend_field", value);
			}
		}

		// Token: 0x17000235 RID: 565
		// (get) Token: 0x0600080D RID: 2061 RVA: 0x000239A6 File Offset: 0x00021BA6
		// (set) Token: 0x0600080E RID: 2062 RVA: 0x000239B3 File Offset: 0x00021BB3
		public string CorresponKey
		{
			get
			{
				return base.GetAttribute("correspon_key");
			}
			set
			{
				this.AttributeChanged("correspon_key", value);
			}
		}

		// Token: 0x17000236 RID: 566
		// (get) Token: 0x0600080F RID: 2063 RVA: 0x000239C1 File Offset: 0x00021BC1
		// (set) Token: 0x06000810 RID: 2064 RVA: 0x000239CE File Offset: 0x00021BCE
		public string LangTable
		{
			get
			{
				return base.GetAttribute("lang_table");
			}
			set
			{
				this.AttributeChanged("lang_table", value);
				this.OnPropertyChanged("Columns");
			}
		}

		// Token: 0x17000237 RID: 567
		// (get) Token: 0x06000811 RID: 2065 RVA: 0x000239E7 File Offset: 0x00021BE7
		// (set) Token: 0x06000812 RID: 2066 RVA: 0x000239F4 File Offset: 0x00021BF4
		public string LangFK
		{
			get
			{
				return base.GetAttribute("lang_fk");
			}
			set
			{
				this.AttributeChanged("lang_fk", value);
			}
		}

		// Token: 0x17000238 RID: 568
		// (get) Token: 0x06000813 RID: 2067 RVA: 0x00023A02 File Offset: 0x00021C02
		// (set) Token: 0x06000814 RID: 2068 RVA: 0x00023A0F File Offset: 0x00021C0F
		public string LangDlang
		{
			get
			{
				return base.GetAttribute("lang_dlang");
			}
			set
			{
				this.AttributeChanged("lang_dlang", value);
			}
		}

		// Token: 0x17000239 RID: 569
		// (get) Token: 0x06000815 RID: 2069 RVA: 0x00023A1D File Offset: 0x00021C1D
		// (set) Token: 0x06000816 RID: 2070 RVA: 0x00023A2A File Offset: 0x00021C2A
		public string LangRTN
		{
			get
			{
				return base.GetAttribute("lang_rtn");
			}
			set
			{
				this.AttributeChanged("lang_rtn", value);
			}
		}

		// Token: 0x1700023A RID: 570
		// (get) Token: 0x06000817 RID: 2071 RVA: 0x00023A38 File Offset: 0x00021C38
		public IEnumerable<XElement> Tables
		{
			get
			{
				return TableColumnHelper.GetTables();
			}
		}

		// Token: 0x1700023B RID: 571
		// (get) Token: 0x06000818 RID: 2072 RVA: 0x00023A40 File Offset: 0x00021C40
		public IEnumerable<XElement> Columns
		{
			get
			{
				if (string.IsNullOrEmpty(this.LangTable))
				{
					return null;
				}
				XElement xelement = TableColumnHelper.FindTableColumns(this.LangTable);
				if (xelement == null)
				{
					return null;
				}
				return xelement.Elements("column");
			}
		}

		// Token: 0x06000819 RID: 2073 RVA: 0x00023A80 File Offset: 0x00021C80
		protected override void AttributeChanged(string key, string newValue)
		{
			if (base.IsCited)
			{
				return;
			}
			string attribute = base.GetAttribute(key);
			if (attribute == newValue)
			{
				return;
			}
			SpecAttributeUndoRedoCommand specAttributeUndoRedoCommand = new SpecAttributeUndoRedoCommand(this);
			if (key != null)
			{
				if (!(key == "depend_field"))
				{
					if (key == "lang_table")
					{
						specAttributeUndoRedoCommand.AddAttributeChanged("lang_fk", this.LangFK, "");
						specAttributeUndoRedoCommand.AddAttributeChanged("lang_dlang", this.LangDlang, "");
						specAttributeUndoRedoCommand.AddAttributeChanged("lang_rtn", this.LangRTN, "");
					}
				}
				else
				{
					string[] array = newValue.Split(new char[] { '.' });
					if (array.Count<string>() == 2)
					{
						string text = array[0];
						string text2 = array[1];
						XElement xelement = TableColumnHelper.FindMultiLangField(text, text2);
						if (xelement != null)
						{
							string value = xelement.Attribute("correspon_key").Value;
							string value2 = xelement.Attribute("lang_table").Value;
							string value3 = xelement.Attribute("lang_fk").Value;
							string value4 = xelement.Attribute("lang_dlang").Value;
							string value5 = xelement.Attribute("lang_rtn").Value;
							specAttributeUndoRedoCommand.AddAttributeChanged("correspon_key", this.CorresponKey, value);
							specAttributeUndoRedoCommand.AddAttributeChanged("lang_table", this.LangTable, value2);
							specAttributeUndoRedoCommand.AddAttributeChanged("lang_fk", this.LangFK, value3);
							specAttributeUndoRedoCommand.AddAttributeChanged("lang_dlang", this.LangDlang, value4);
							specAttributeUndoRedoCommand.AddAttributeChanged("lang_rtn", this.LangRTN, value5);
						}
					}
				}
			}
			specAttributeUndoRedoCommand.AddAttributeChanged(key, base.GetAttribute(key), newValue);
			specAttributeUndoRedoCommand.Execute();
			base.AttributeChanged(key, newValue);
		}

		// Token: 0x1700023C RID: 572
		// (get) Token: 0x0600081A RID: 2074 RVA: 0x00023CAC File Offset: 0x00021EAC
		public override XElement CitedSpec
		{
			get
			{
				if (SettingManager.Get().GetTzpManger(base.ProgramKey).IsStandardProgram)
				{
					return null;
				}
				XElement citeSTD = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.CiteSTD;
				if (citeSTD == null)
				{
					return null;
				}
				return (from e in citeSTD.Element("multi_lang").Elements()
					where e.Attribute("name").Value == this.Name && e.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select e).FirstOrDefault<XElement>();
			}
		}

		// Token: 0x1700023D RID: 573
		// (get) Token: 0x0600081B RID: 2075 RVA: 0x00023D1F File Offset: 0x00021F1F
		public string Error
		{
			get
			{
				return string.Empty;
			}
		}

		// Token: 0x1700023E RID: 574
		public string this[string columnName]
		{
			get
			{
				string text = null;
				switch (columnName)
				{
				case "Name":
					if (string.IsNullOrEmpty(base.GetAttribute("name")))
					{
						text = "this field is required";
					}
					break;
				case "LangTable":
					if (string.IsNullOrEmpty(base.GetAttribute("lang_table")))
					{
						text = "this field is required";
					}
					break;
				case "DependField":
					if (string.IsNullOrEmpty(base.GetAttribute("depend_field")))
					{
						text = "this field is required";
					}
					break;
				case "CorresponKey":
					if (string.IsNullOrEmpty(base.GetAttribute("correspon_key")))
					{
						text = "this field is required";
					}
					break;
				case "LangFK":
					if (string.IsNullOrEmpty(base.GetAttribute("lang_fk")))
					{
						text = "this field is required";
					}
					break;
				case "LangDlang":
					if (string.IsNullOrEmpty(base.GetAttribute("lang_dlang")))
					{
						text = "this field is required";
					}
					break;
				case "LangRTN":
					if (string.IsNullOrEmpty(base.GetAttribute("lang_rtn")))
					{
						text = "this field is required";
					}
					break;
				}
				return text;
			}
		}

		// Token: 0x0600081D RID: 2077 RVA: 0x00023EA8 File Offset: 0x000220A8
		public static SpecMultiLangNode Create(SpecificationInfo info, string name)
		{
			XElement xelement = new XElement("mfield", new object[]
			{
				new XAttribute("src", info.Env),
				new XAttribute("ver", info.Ver),
				new XAttribute("name", name),
				new XAttribute("depend_field", ""),
				new XAttribute("correspon_key", ""),
				new XAttribute("lang_table", ""),
				new XAttribute("lang_fk", ""),
				new XAttribute("lang_dlang", ""),
				new XAttribute("lang_rtn", ""),
				new XAttribute("cite_std", "N"),
				new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE))
			});
			return new SpecMultiLangNode(info.Key, xelement);
		}

		// Token: 0x0600081E RID: 2078 RVA: 0x00023FDD File Offset: 0x000221DD
		public static SpecMultiLangNode Create(PackageKey key, XElement source)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecMultiLangNode(key, source);
		}

		// Token: 0x0600081F RID: 2079 RVA: 0x00023FEB File Offset: 0x000221EB
		public static SpecMultiLangNode Create(PackageKey key, XElement source, string src)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecMultiLangNode(key, source, src);
		}
	}
}
