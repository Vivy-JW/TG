using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Xml.Linq;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200012C RID: 300
	public class SpecReferenceNode : AbstractSpecNode, IDataErrorInfo
	{
		// Token: 0x06000A96 RID: 2710 RVA: 0x00034110 File Offset: 0x00032310
		public SpecReferenceNode(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x06000A97 RID: 2711 RVA: 0x00034125 File Offset: 0x00032325
		public SpecReferenceNode(PackageKey key, XElement source, string src)
			: base(key, source, src)
		{
		}

		// Token: 0x170002C2 RID: 706
		// (get) Token: 0x06000A98 RID: 2712 RVA: 0x0003413B File Offset: 0x0003233B
		// (set) Token: 0x06000A99 RID: 2713 RVA: 0x00034148 File Offset: 0x00032348
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

		// Token: 0x170002C3 RID: 707
		// (get) Token: 0x06000A9A RID: 2714 RVA: 0x00034156 File Offset: 0x00032356
		// (set) Token: 0x06000A9B RID: 2715 RVA: 0x00034163 File Offset: 0x00032363
		public string RefTable
		{
			get
			{
				return base.GetAttribute("ref_table");
			}
			set
			{
				this.AttributeChanged("ref_table", value);
				this.OnPropertyChanged("Columns");
			}
		}

		// Token: 0x170002C4 RID: 708
		// (get) Token: 0x06000A9C RID: 2716 RVA: 0x0003417C File Offset: 0x0003237C
		// (set) Token: 0x06000A9D RID: 2717 RVA: 0x00034189 File Offset: 0x00032389
		public string RefFk
		{
			get
			{
				return base.GetAttribute("ref_fk");
			}
			set
			{
				this.AttributeChanged("ref_fk", value);
			}
		}

		// Token: 0x170002C5 RID: 709
		// (get) Token: 0x06000A9E RID: 2718 RVA: 0x00034197 File Offset: 0x00032397
		// (set) Token: 0x06000A9F RID: 2719 RVA: 0x000341A4 File Offset: 0x000323A4
		public string RefDlang
		{
			get
			{
				return base.GetAttribute("ref_dlang");
			}
			set
			{
				this.AttributeChanged("ref_dlang", value);
			}
		}

		// Token: 0x170002C6 RID: 710
		// (get) Token: 0x06000AA0 RID: 2720 RVA: 0x000341B2 File Offset: 0x000323B2
		// (set) Token: 0x06000AA1 RID: 2721 RVA: 0x000341BF File Offset: 0x000323BF
		public string RefRtn
		{
			get
			{
				return base.GetAttribute("ref_rtn");
			}
			set
			{
				this.AttributeChanged("ref_rtn", value);
			}
		}

		// Token: 0x170002C7 RID: 711
		// (get) Token: 0x06000AA2 RID: 2722 RVA: 0x000341CD File Offset: 0x000323CD
		// (set) Token: 0x06000AA3 RID: 2723 RVA: 0x000341DA File Offset: 0x000323DA
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

		// Token: 0x170002C8 RID: 712
		// (get) Token: 0x06000AA4 RID: 2724 RVA: 0x000341E8 File Offset: 0x000323E8
		// (set) Token: 0x06000AA5 RID: 2725 RVA: 0x000341F8 File Offset: 0x000323F8
		public new string Name
		{
			get
			{
				return base.GetAttribute("name");
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				if (string.IsNullOrEmpty(value))
				{
					throw new Exception("'name' field is required");
				}
				string value2 = base.Source.Attribute("name").Value;
				if (value.Equals(base.Source.Attribute("name").Value))
				{
					return;
				}
				if (Regex.IsMatch(value, this.LegalNameRgex))
				{
					throw new FormatException(Application.Current.FindResource("Message_ActionNameSyntaxIncorrent") as string);
				}
				if (SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.IsExists(value))
				{
					return;
				}
				if (!value.EndsWith("_desc"))
				{
					DocumentErrorsEventArgs e = new DocumentErrorsEventArgs();
					e.ProgramKey = base.ProgramKey;
					e.SourceType = base.ProgramKey.PackType;
					e.ErrorType = ErrorsType.WARNING;
					e.Time = DateTime.Now;
					e.Key = this.Name;
					e.Description = Application.Current.FindResource("Message_FieldNameFormat") as string;
					EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e);
				}
				RenameUndoRedoCommand renameUndoRedoCommand = new RenameUndoRedoCommand(base.ProgramKey, this.Name, value);
				renameUndoRedoCommand.Execute();
				if (SettingManager.Get().GetUndoRedoManager(base.ProgramKey) != null)
				{
					SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddUndo(renameUndoRedoCommand);
				}
			}
		}

		// Token: 0x170002C9 RID: 713
		// (get) Token: 0x06000AA6 RID: 2726 RVA: 0x0003435E File Offset: 0x0003255E
		public IEnumerable<XElement> Tables
		{
			get
			{
				return TableColumnHelper.GetTables();
			}
		}

		// Token: 0x170002CA RID: 714
		// (get) Token: 0x06000AA7 RID: 2727 RVA: 0x00034368 File Offset: 0x00032568
		public IEnumerable<XElement> Columns
		{
			get
			{
				if (string.IsNullOrEmpty(this.RefTable))
				{
					return null;
				}
				XElement xelement = TableColumnHelper.FindTableColumns(this.RefTable);
				if (xelement == null)
				{
					return null;
				}
				return xelement.Elements("column");
			}
		}

		// Token: 0x06000AA8 RID: 2728 RVA: 0x000343A8 File Offset: 0x000325A8
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
					if (key == "ref_table")
					{
						specAttributeUndoRedoCommand.AddAttributeChanged("ref_fk", this.RefFk, "");
						specAttributeUndoRedoCommand.AddAttributeChanged("ref_dlang", this.RefDlang, "");
						specAttributeUndoRedoCommand.AddAttributeChanged("ref_rtn", this.RefRtn, "");
					}
				}
				else if (!(newValue == ""))
				{
					string[] array = newValue.Split(new char[] { '.' });
					if (array.Count<string>() == 2)
					{
						string text = array[0];
						string text2 = array[1];
						XElement xelement = TableColumnHelper.FindRefField(text, text2);
						if (xelement != null)
						{
							string value = xelement.Attribute("correspon_key").Value;
							string value2 = xelement.Attribute("ref_table").Value;
							string value3 = xelement.Attribute("ref_fk").Value;
							string value4 = xelement.Attribute("ref_dlang").Value;
							string value5 = xelement.Attribute("ref_rtn").Value;
							specAttributeUndoRedoCommand.AddAttributeChanged("correspon_key", this.CorresponKey, value);
							specAttributeUndoRedoCommand.AddAttributeChanged("ref_table", this.RefTable, value2);
							specAttributeUndoRedoCommand.AddAttributeChanged("ref_fk", this.RefFk, value3);
							specAttributeUndoRedoCommand.AddAttributeChanged("ref_dlang", this.RefDlang, value4);
							specAttributeUndoRedoCommand.AddAttributeChanged("ref_rtn", this.RefRtn, value5);
						}
					}
					string text3 = ((array.Count<string>() == 2) ? array[1] : newValue);
					string text4 = string.Format("{0}_desc", text3);
					string name = this.Name;
					if (name != text4)
					{
						text4 = ComponentFactory.GetNewName(text4, base.ProgramKey);
						specAttributeUndoRedoCommand.AddAttributeChanged("name", name, text4);
					}
				}
			}
			specAttributeUndoRedoCommand.AddAttributeChanged(key, base.GetAttribute(key), newValue);
			specAttributeUndoRedoCommand.Execute();
			base.AttributeChanged(key, newValue);
		}

		// Token: 0x170002CB RID: 715
		// (get) Token: 0x06000AA9 RID: 2729 RVA: 0x00034634 File Offset: 0x00032834
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
				return (from e in citeSTD.Element("ref_field").Elements()
					where e.Attribute("name").Value == this.Name && e.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select e).FirstOrDefault<XElement>();
			}
		}

		// Token: 0x170002CC RID: 716
		// (get) Token: 0x06000AAA RID: 2730 RVA: 0x000346A7 File Offset: 0x000328A7
		public string Error
		{
			get
			{
				return string.Empty;
			}
		}

		// Token: 0x170002CD RID: 717
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
							if (!(columnName == "CorresponKey"))
							{
								if (!(columnName == "RefTable"))
								{
									if (!(columnName == "RefFk"))
									{
										if (columnName == "RefRtn")
										{
											if (string.IsNullOrEmpty(base.GetAttribute("ref_rtn")))
											{
												text = "this field is required";
											}
										}
									}
									else if (string.IsNullOrEmpty(base.GetAttribute("ref_fk")))
									{
										text = "this field is required";
									}
								}
								else if (string.IsNullOrEmpty(base.GetAttribute("ref_table")))
								{
									text = "this field is required";
								}
							}
							else if (string.IsNullOrEmpty(base.GetAttribute("correspon_key")))
							{
								text = "this field is required";
							}
						}
						else if (string.IsNullOrEmpty(base.GetAttribute("depend_field")))
						{
							text = "this field is required";
						}
					}
					else if (string.IsNullOrEmpty(base.GetAttribute("name")))
					{
						text = "this field is required";
					}
				}
				return text;
			}
		}

		// Token: 0x06000AAC RID: 2732 RVA: 0x000347C4 File Offset: 0x000329C4
		public static SpecReferenceNode Create(SpecificationInfo info, string name)
		{
			XElement xelement = new XElement("rfield", new object[]
			{
				new XAttribute("src", info.Env),
				new XAttribute("ver", info.Ver),
				new XAttribute("name", name),
				new XAttribute("depend_field", ""),
				new XAttribute("correspon_key", ""),
				new XAttribute("ref_table", ""),
				new XAttribute("ref_fk", ""),
				new XAttribute("ref_dlang", ""),
				new XAttribute("ref_rtn", ""),
				new XAttribute("cite_std", "N"),
				new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE))
			});
			return new SpecReferenceNode(info.Key, xelement);
		}

		// Token: 0x06000AAD RID: 2733 RVA: 0x000348F9 File Offset: 0x00032AF9
		public static SpecReferenceNode Create(PackageKey key, XElement source)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecReferenceNode(key, source);
		}

		// Token: 0x06000AAE RID: 2734 RVA: 0x00034907 File Offset: 0x00032B07
		public static SpecReferenceNode Create(PackageKey key, XElement source, string src)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecReferenceNode(key, source, src);
		}

		// Token: 0x040003F9 RID: 1017
		private readonly string LegalNameRgex = "[^a-zA-Z0-9_]";
	}
}
