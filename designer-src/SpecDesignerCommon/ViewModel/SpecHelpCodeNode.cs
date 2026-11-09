using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000A6 RID: 166
	public class SpecHelpCodeNode : AbstractSpecNode
	{
		// Token: 0x060006FF RID: 1791 RVA: 0x0001F487 File Offset: 0x0001D687
		public SpecHelpCodeNode(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x06000700 RID: 1792 RVA: 0x0001F491 File Offset: 0x0001D691
		public SpecHelpCodeNode(PackageKey key, XElement source, string src)
			: base(key, source, src)
		{
		}

		// Token: 0x170001FE RID: 510
		// (get) Token: 0x06000701 RID: 1793 RVA: 0x0001F49C File Offset: 0x0001D69C
		// (set) Token: 0x06000702 RID: 1794 RVA: 0x0001F4A9 File Offset: 0x0001D6A9
		public string HelpTable
		{
			get
			{
				return base.GetAttribute("help_table");
			}
			set
			{
				this.AttributeChanged("help_table", value);
			}
		}

		// Token: 0x170001FF RID: 511
		// (get) Token: 0x06000703 RID: 1795 RVA: 0x0001F4B7 File Offset: 0x0001D6B7
		// (set) Token: 0x06000704 RID: 1796 RVA: 0x0001F4C4 File Offset: 0x0001D6C4
		public string HelpFind
		{
			get
			{
				return base.GetAttribute("help_find");
			}
			set
			{
				this.AttributeChanged("help_find", value);
			}
		}

		// Token: 0x17000200 RID: 512
		// (get) Token: 0x06000705 RID: 1797 RVA: 0x0001F4D2 File Offset: 0x0001D6D2
		// (set) Token: 0x06000706 RID: 1798 RVA: 0x0001F4DF File Offset: 0x0001D6DF
		public string HelpLang
		{
			get
			{
				return base.GetAttribute("help_dlang");
			}
			set
			{
				this.AttributeChanged("help_dlang", value);
			}
		}

		// Token: 0x17000201 RID: 513
		// (get) Token: 0x06000707 RID: 1799 RVA: 0x0001F4ED File Offset: 0x0001D6ED
		// (set) Token: 0x06000708 RID: 1800 RVA: 0x0001F4FA File Offset: 0x0001D6FA
		public string HelpField
		{
			get
			{
				return base.GetAttribute("help_field");
			}
			set
			{
				this.AttributeChanged("help_field", value);
			}
		}

		// Token: 0x17000202 RID: 514
		// (get) Token: 0x06000709 RID: 1801 RVA: 0x0001F508 File Offset: 0x0001D708
		// (set) Token: 0x0600070A RID: 1802 RVA: 0x0001F516 File Offset: 0x0001D716
		public string MappingWidget
		{
			get
			{
				this.GetMappingWidgetMixValue();
				return this._mappingWidget;
			}
			set
			{
				this._mappingWidget = value;
				this.AttributeChanged("mapping_widget", this.SetMappingWidgetMixValue());
			}
		}

		// Token: 0x17000203 RID: 515
		// (get) Token: 0x0600070B RID: 1803 RVA: 0x0001F530 File Offset: 0x0001D730
		// (set) Token: 0x0600070C RID: 1804 RVA: 0x0001F53D File Offset: 0x0001D73D
		public string HelpWC
		{
			get
			{
				return base.GetAttribute("help_wc");
			}
			set
			{
				this.AttributeChanged("help_wc", value);
			}
		}

		// Token: 0x17000204 RID: 516
		// (get) Token: 0x0600070D RID: 1805 RVA: 0x0001F54B File Offset: 0x0001D74B
		// (set) Token: 0x0600070E RID: 1806 RVA: 0x0001F559 File Offset: 0x0001D759
		public string ReturningTableField
		{
			get
			{
				this.GetMappingWidgetMixValue();
				return this._returningTableField;
			}
			set
			{
				this._returningTableField = value;
				this.AttributeChanged("mapping_widget", this.SetMappingWidgetMixValue());
			}
		}

		// Token: 0x0600070F RID: 1807 RVA: 0x0001F574 File Offset: 0x0001D774
		protected void GetMappingWidgetMixValue()
		{
			string attribute = base.GetAttribute("mapping_widget");
			if (!string.IsNullOrEmpty(attribute))
			{
				string[] array = attribute.Split(new char[] { ',' });
				List<string> list = new List<string>();
				List<string> list2 = new List<string>();
				foreach (string text in array)
				{
					string[] array3 = text.Split(new char[] { ':' });
					if (!string.IsNullOrEmpty(array3[0]))
					{
						list.Add(array3[0]);
					}
					if (array3.Length > 1 && !string.IsNullOrEmpty(array3[1]))
					{
						list2.Add(array3[1]);
					}
				}
				this._returningTableField = string.Join(",", list.ToArray());
				this._mappingWidget = string.Join(",", list2.ToArray());
			}
		}

		// Token: 0x06000710 RID: 1808 RVA: 0x0001F650 File Offset: 0x0001D850
		protected string SetMappingWidgetMixValue()
		{
			StringBuilder stringBuilder = new StringBuilder();
			string[] array = null;
			string[] array2 = null;
			if (!string.IsNullOrEmpty(this._returningTableField))
			{
				array = this._returningTableField.Split(new char[] { ',' });
			}
			if (!string.IsNullOrEmpty(this._mappingWidget))
			{
				array2 = this._mappingWidget.Split(new char[] { ',' });
			}
			int num = 0;
			if (array != null)
			{
				num = array.Length;
			}
			if (array2 != null && (array2.Length > num || num == 0))
			{
				num = array2.Length;
			}
			for (int i = 0; i < num; i++)
			{
				if (i > 0)
				{
					stringBuilder.Append(',');
				}
				if (array != null && i < array.Length)
				{
					stringBuilder.Append(array[i]);
				}
				stringBuilder.Append(':');
				if (array2 != null && i < array2.Length)
				{
					stringBuilder.Append(array2[i]);
				}
			}
			return stringBuilder.ToString();
		}

		// Token: 0x06000711 RID: 1809 RVA: 0x0001F72C File Offset: 0x0001D92C
		public bool ValidMappingWidgetMixValue()
		{
			string[] array = null;
			string[] array2 = null;
			bool flag;
			if (string.IsNullOrEmpty(this._returningTableField) && string.IsNullOrEmpty(this._mappingWidget))
			{
				flag = true;
			}
			else
			{
				if (!string.IsNullOrEmpty(this._returningTableField))
				{
					array = this._returningTableField.Split(new char[] { ',' });
				}
				if (!string.IsNullOrEmpty(this._mappingWidget))
				{
					array2 = this._mappingWidget.Split(new char[] { ',' });
				}
				if (array != null && array2 != null)
				{
					if (array.Length == array2.Length)
					{
						int num = 0;
						int num2 = 0;
						for (int i = 0; i < array.Length; i++)
						{
							if (!string.IsNullOrEmpty(array[i]))
							{
								num++;
							}
							if (!string.IsNullOrEmpty(array2[i]))
							{
								num2++;
							}
						}
						flag = num == num2;
					}
					else
					{
						flag = false;
					}
				}
				else
				{
					flag = false;
				}
			}
			return flag;
		}

		// Token: 0x06000712 RID: 1810 RVA: 0x0001F808 File Offset: 0x0001DA08
		protected override void AttributeChanged(string key, string newValue)
		{
			if (base.IsCited)
			{
				return;
			}
			string attribute = base.GetAttribute(key);
			if (string.IsNullOrEmpty(newValue) && string.IsNullOrEmpty(attribute))
			{
				return;
			}
			if (attribute == newValue)
			{
				return;
			}
			SpecAttributeUndoRedoCommand specAttributeUndoRedoCommand = new SpecAttributeUndoRedoCommand(this);
			if (key != null)
			{
				if (!(key == "help_table"))
				{
					if (key == "mapping_widget")
					{
						specAttributeUndoRedoCommand.AddAttributeChanged("mapping_widget", this.SetMappingWidgetMixValue(), "");
						goto IL_0116;
					}
				}
				else
				{
					specAttributeUndoRedoCommand.AddAttributeChanged("help_table", this.HelpTable, newValue);
					if ("" == newValue)
					{
						specAttributeUndoRedoCommand.AddAttributeChanged("help_find", this.HelpFind, "");
						specAttributeUndoRedoCommand.AddAttributeChanged("help_dlang", this.HelpLang, "");
						specAttributeUndoRedoCommand.AddAttributeChanged("help_field", this.HelpField, "");
						specAttributeUndoRedoCommand.AddAttributeChanged("mapping_widget", this.SetMappingWidgetMixValue(), "");
						specAttributeUndoRedoCommand.AddAttributeChanged("help_wc", this.HelpWC, "");
						goto IL_0116;
					}
					goto IL_0116;
				}
			}
			specAttributeUndoRedoCommand.AddAttributeChanged(key, base.GetAttribute(key), newValue);
			IL_0116:
			specAttributeUndoRedoCommand.Execute();
			base.AttributeChanged(key, newValue);
		}

		// Token: 0x17000205 RID: 517
		// (get) Token: 0x06000713 RID: 1811 RVA: 0x0001F994 File Offset: 0x0001DB94
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
				return (from e in citeSTD.Descendants("hfield")
					where e.Attribute("name").Value == this.Name && e.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select e).FirstOrDefault<XElement>();
			}
		}

		// Token: 0x06000714 RID: 1812 RVA: 0x0001FA04 File Offset: 0x0001DC04
		public static SpecHelpCodeNode Create(SpecificationInfo info, string name)
		{
			XElement xelement = new XElement("hfield", new object[]
			{
				new XAttribute("src", info.Env),
				new XAttribute("ver", info.Ver),
				new XAttribute("name", name),
				new XAttribute("help_table", ""),
				new XAttribute("help_find", ""),
				new XAttribute("help_dlang", ""),
				new XAttribute("help_field", ""),
				new XAttribute("mapping_widget", ""),
				new XAttribute("help_wc", ""),
				new XAttribute("cite_std", "N"),
				new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE))
			});
			return new SpecHelpCodeNode(info.Key, xelement);
		}

		// Token: 0x06000715 RID: 1813 RVA: 0x0001FB39 File Offset: 0x0001DD39
		public static SpecHelpCodeNode Create(PackageKey key, XElement source)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecHelpCodeNode(key, source);
		}

		// Token: 0x06000716 RID: 1814 RVA: 0x0001FB47 File Offset: 0x0001DD47
		public static SpecHelpCodeNode Create(PackageKey key, XElement source, string src)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecHelpCodeNode(key, source, src);
		}

		// Token: 0x04000286 RID: 646
		private string _mappingWidget;

		// Token: 0x04000287 RID: 647
		private string _returningTableField;
	}
}
