using System;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000A3 RID: 163
	public class SpecProgRelNode : AbstractSpecNode, IDataErrorInfo
	{
		// Token: 0x060006D3 RID: 1747 RVA: 0x0001EB08 File Offset: 0x0001CD08
		public SpecProgRelNode(PackageKey key, XElement source)
			: base(key, source)
		{
			this.Programs = new ObservableCollection<ProgRelProgram>();
			foreach (XElement xelement in source.Elements("program"))
			{
				this.Programs.Add(new ProgRelProgram(this, xelement));
			}
		}

		// Token: 0x060006D4 RID: 1748 RVA: 0x0001EB80 File Offset: 0x0001CD80
		public SpecProgRelNode(PackageKey key, XElement source, string src)
			: base(key, source, src)
		{
			this.Programs = new ObservableCollection<ProgRelProgram>();
			foreach (XElement xelement in source.Elements("program"))
			{
				this.Programs.Add(new ProgRelProgram(this, xelement));
			}
		}

		// Token: 0x170001F1 RID: 497
		// (get) Token: 0x060006D5 RID: 1749 RVA: 0x0001EBF8 File Offset: 0x0001CDF8
		// (set) Token: 0x060006D6 RID: 1750 RVA: 0x0001EC08 File Offset: 0x0001CE08
		public string DependField
		{
			get
			{
				return base.GetAttribute("depend_field");
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				if (string.IsNullOrEmpty(base.GetAttribute("depend_field")))
				{
					FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.FindNodeByName(base.GetAttribute("depend_field"));
					if (formSpecModel != null && formSpecModel.SpecField != null)
					{
						formSpecModel.SpecField.OnPropertyChanged("Status");
					}
				}
				if (!string.IsNullOrEmpty(value))
				{
					FormSpecModel formSpecModel2 = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.FindNodeByName(value);
					if (formSpecModel2 != null && formSpecModel2.SpecField != null)
					{
						formSpecModel2.SpecField.OnPropertyChanged("Status");
					}
				}
				FormSpecModel formSpecModel3 = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.FindNodeByName(this.Name);
				if (formSpecModel3 != null && formSpecModel3.SpecField != null)
				{
					formSpecModel3.SpecField.OnPropertyChanged("Status");
				}
				this.AttributeChanged("depend_field", value);
			}
		}

		// Token: 0x170001F2 RID: 498
		// (get) Token: 0x060006D7 RID: 1751 RVA: 0x0001ECF8 File Offset: 0x0001CEF8
		// (set) Token: 0x060006D8 RID: 1752 RVA: 0x0001ED00 File Offset: 0x0001CF00
		public ObservableCollection<ProgRelProgram> Programs { get; set; }

		// Token: 0x060006D9 RID: 1753 RVA: 0x0001ED0C File Offset: 0x0001CF0C
		protected override void AttributeChanged(string key, string newValue)
		{
			SpecAttributeUndoRedoCommand specAttributeUndoRedoCommand = new SpecAttributeUndoRedoCommand(this);
			specAttributeUndoRedoCommand.AddAttributeChanged(key, base.GetAttribute(key), newValue);
			specAttributeUndoRedoCommand.Execute();
			base.AttributeChanged(key, newValue);
		}

		// Token: 0x170001F3 RID: 499
		// (get) Token: 0x060006DA RID: 1754 RVA: 0x0001ED98 File Offset: 0x0001CF98
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
				return (from e in citeSTD.Element("prog_rel").Elements()
					where e.Attribute("name").Value == this.Name && e.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select e).FirstOrDefault<XElement>();
			}
		}

		// Token: 0x060006DB RID: 1755 RVA: 0x0001EE0B File Offset: 0x0001D00B
		public void Remove(ProgRelProgram selectedProgram)
		{
			this.Programs.Remove(selectedProgram);
			this.ResetChildren();
		}

		// Token: 0x060006DC RID: 1756 RVA: 0x0001EE20 File Offset: 0x0001D020
		public void Appand(ProgRelProgram program)
		{
			this.Programs.Add(program);
			this.ResetChildren();
		}

		// Token: 0x060006DD RID: 1757 RVA: 0x0001EE34 File Offset: 0x0001D034
		private void ResetChildren()
		{
			base.Source.RemoveNodes();
			foreach (ProgRelProgram progRelProgram in this.Programs)
			{
				base.Source.Add(progRelProgram.Source);
			}
			FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.FindNodeByName(this.Name);
			if (formSpecModel != null && formSpecModel.SpecField != null)
			{
				formSpecModel.SpecField.OnPropertyChanged("Status");
			}
			this.OnPropertyChanged("Programs");
		}

		// Token: 0x060006DE RID: 1758 RVA: 0x0001EEE0 File Offset: 0x0001D0E0
		internal int getMaxOrder()
		{
			int num = 0;
			foreach (ProgRelProgram progRelProgram in this.Programs)
			{
				short num2 = -1;
				if (short.TryParse(progRelProgram.Order, out num2))
				{
					num = Math.Max(num, (int)num2);
				}
			}
			return num;
		}

		// Token: 0x060006DF RID: 1759 RVA: 0x0001EF44 File Offset: 0x0001D144
		public new XElement ToXml()
		{
			if (SpecStatus.CREATE == this.Status)
			{
				return null;
			}
			base.SetStatusToSource();
			return base.Source;
		}

		// Token: 0x060006E0 RID: 1760 RVA: 0x0001EF5D File Offset: 0x0001D15D
		public override string ToString()
		{
			base.SetStatusToSource();
			return base.Source.ToString();
		}

		// Token: 0x170001F4 RID: 500
		// (get) Token: 0x060006E1 RID: 1761 RVA: 0x0001EF70 File Offset: 0x0001D170
		public string Error
		{
			get
			{
				return string.Empty;
			}
		}

		// Token: 0x170001F5 RID: 501
		public string this[string columnName]
		{
			get
			{
				string text = null;
				if (columnName != null)
				{
					if (!(columnName == "Name"))
					{
						if (!(columnName == "DependField"))
						{
							if (columnName == "Program")
							{
								if (string.IsNullOrEmpty(base.GetAttribute("program")))
								{
									text = "'program' field is required";
								}
							}
						}
						else if (string.IsNullOrEmpty(base.GetAttribute("depend_field")) && this.Programs.Count > 0)
						{
							text = "'depend_field' field is required";
						}
					}
					else if (string.IsNullOrEmpty(base.GetAttribute("name")))
					{
						text = "'name' field is required";
					}
				}
				return text;
			}
		}

		// Token: 0x060006E3 RID: 1763 RVA: 0x0001F014 File Offset: 0x0001D214
		public static SpecProgRelNode Create(SpecificationInfo info, string name)
		{
			XElement xelement = new XElement("pfield", new object[]
			{
				new XAttribute("src", info.Env),
				new XAttribute("ver", info.Ver),
				new XAttribute("name", name),
				new XAttribute("depend_field", ""),
				new XAttribute("program", ""),
				new XAttribute("type", "1"),
				new XAttribute("cite_std", "N"),
				new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE))
			});
			return new SpecProgRelNode(info.Key, xelement);
		}

		// Token: 0x060006E4 RID: 1764 RVA: 0x0001F101 File Offset: 0x0001D301
		public static SpecProgRelNode Create(PackageKey key, XElement source)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecProgRelNode(key, source);
		}

		// Token: 0x060006E5 RID: 1765 RVA: 0x0001F10F File Offset: 0x0001D30F
		public static SpecProgRelNode Create(PackageKey key, XElement source, string src)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecProgRelNode(key, source, src);
		}
	}
}
