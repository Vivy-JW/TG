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
	// Token: 0x0200007C RID: 124
	public class SpecFieldNode : AbstractSpecNode, IDataErrorInfo
	{
		// Token: 0x060004AB RID: 1195 RVA: 0x00014D29 File Offset: 0x00012F29
		public SpecFieldNode(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x060004AC RID: 1196 RVA: 0x00014D33 File Offset: 0x00012F33
		public SpecFieldNode(PackageKey key, XElement source, string src)
			: base(key, source, src)
		{
		}

		// Token: 0x060004AD RID: 1197 RVA: 0x00014D40 File Offset: 0x00012F40
		internal new void SetName(FormSpecModel model, string newName)
		{
			if (Regex.IsMatch(newName, "(?![a-zA-Z0-9_\\.]+)."))
			{
				throw new Exception(Application.Current.FindResource("Message_FieldNameSyntaxIncorrent") as string);
			}
			if (model == null)
			{
				return;
			}
			if (model.GeneroComponent != null)
			{
				bool flag = model.GeneroComponent.Parent != null;
				string text = ((!flag) ? null : model.GeneroComponent.Parent.NodeName);
				string text2 = ((!flag) ? null : model.GeneroComponent.Parent.Name);
				string text3 = string.Format("{0}.{1}", this.Table, this.Column);
				if (newName != text3)
				{
					string[] array = new string[] { "Tree", "ScrollGrid", "Table" };
					if (flag && (text2 == "s_browse" || (SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.GetClass() == "Q" && array.Contains(text))))
					{
						if (!newName.StartsWith("b_"))
						{
							DocumentErrorsEventArgs e = new DocumentErrorsEventArgs();
							e.ProgramKey = base.ProgramKey;
							e.SourceType = base.ProgramKey.PackType;
							e.ErrorType = ErrorsType.WARNING;
							e.Time = DateTime.Now;
							e.Key = this.Name;
							e.Description = Application.Current.FindResource("Message_DetailFieldNamePrefix") as string;
							EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e);
						}
					}
					else if (!newName.StartsWith("l_") || !newName.EndsWith("_desc"))
					{
						DocumentErrorsEventArgs e2 = new DocumentErrorsEventArgs();
						e2.ProgramKey = base.ProgramKey;
						e2.SourceType = base.ProgramKey.PackType;
						e2.ErrorType = ErrorsType.WARNING;
						e2.Time = DateTime.Now;
						e2.Key = this.Name;
						e2.Description = Application.Current.FindResource("Message_FieldNameFormat") as string;
						EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e2);
					}
				}
			}
			base.SetName(model, newName);
		}

		// Token: 0x17000143 RID: 323
		// (get) Token: 0x060004AE RID: 1198 RVA: 0x00014F73 File Offset: 0x00013173
		// (set) Token: 0x060004AF RID: 1199 RVA: 0x00014F80 File Offset: 0x00013180
		public string Req
		{
			get
			{
				return base.GetAttribute("req");
			}
			set
			{
				this.AttributeChanged("req", value);
			}
		}

		// Token: 0x17000144 RID: 324
		// (get) Token: 0x060004B0 RID: 1200 RVA: 0x00014F8E File Offset: 0x0001318E
		// (set) Token: 0x060004B1 RID: 1201 RVA: 0x00014F9B File Offset: 0x0001319B
		public string Items
		{
			get
			{
				return base.GetAttribute("items");
			}
			set
			{
				this.AttributeChanged("items", value);
			}
		}

		// Token: 0x17000145 RID: 325
		// (get) Token: 0x060004B2 RID: 1202 RVA: 0x00014FA9 File Offset: 0x000131A9
		// (set) Token: 0x060004B3 RID: 1203 RVA: 0x00014FB6 File Offset: 0x000131B6
		public string Default
		{
			get
			{
				return base.GetAttribute("default");
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				if (value.Contains("'"))
				{
					throw new Exception("only varchar type field can contain ',\"");
				}
				this.AttributeChanged("default", value);
			}
		}

		// Token: 0x17000146 RID: 326
		// (get) Token: 0x060004B4 RID: 1204 RVA: 0x00014FE5 File Offset: 0x000131E5
		// (set) Token: 0x060004B5 RID: 1205 RVA: 0x00014FF2 File Offset: 0x000131F2
		public string Max
		{
			get
			{
				return base.GetAttribute("max");
			}
			set
			{
				this.AttributeChanged("max", value);
			}
		}

		// Token: 0x17000147 RID: 327
		// (get) Token: 0x060004B6 RID: 1206 RVA: 0x00015000 File Offset: 0x00013200
		// (set) Token: 0x060004B7 RID: 1207 RVA: 0x00015017 File Offset: 0x00013217
		public string Max_Compare
		{
			get
			{
				return base.GetAttribute("max_compare") + " ";
			}
			set
			{
				this.AttributeChanged("max_compare", value.Trim());
			}
		}

		// Token: 0x17000148 RID: 328
		// (get) Token: 0x060004B8 RID: 1208 RVA: 0x0001502A File Offset: 0x0001322A
		// (set) Token: 0x060004B9 RID: 1209 RVA: 0x00015037 File Offset: 0x00013237
		public string Min
		{
			get
			{
				return base.GetAttribute("min");
			}
			set
			{
				this.AttributeChanged("min", value);
			}
		}

		// Token: 0x17000149 RID: 329
		// (get) Token: 0x060004BA RID: 1210 RVA: 0x00015045 File Offset: 0x00013245
		// (set) Token: 0x060004BB RID: 1211 RVA: 0x0001505C File Offset: 0x0001325C
		public string Min_Compare
		{
			get
			{
				return base.GetAttribute("min_compare") + " ";
			}
			set
			{
				this.AttributeChanged("min_compare", value.Trim());
			}
		}

		// Token: 0x1700014A RID: 330
		// (get) Token: 0x060004BC RID: 1212 RVA: 0x0001506F File Offset: 0x0001326F
		// (set) Token: 0x060004BD RID: 1213 RVA: 0x0001507C File Offset: 0x0001327C
		public string IZoom
		{
			get
			{
				return base.GetAttribute("i_zoom");
			}
			set
			{
				this.AttributeChanged("i_zoom", value);
			}
		}

		// Token: 0x1700014B RID: 331
		// (get) Token: 0x060004BE RID: 1214 RVA: 0x0001508A File Offset: 0x0001328A
		// (set) Token: 0x060004BF RID: 1215 RVA: 0x00015097 File Offset: 0x00013297
		public string CZoom
		{
			get
			{
				return base.GetAttribute("c_zoom");
			}
			set
			{
				this.AttributeChanged("c_zoom", value);
			}
		}

		// Token: 0x1700014C RID: 332
		// (get) Token: 0x060004C0 RID: 1216 RVA: 0x000150A5 File Offset: 0x000132A5
		// (set) Token: 0x060004C1 RID: 1217 RVA: 0x000150B2 File Offset: 0x000132B2
		public string ChkRef
		{
			get
			{
				return base.GetAttribute("chk_ref");
			}
			set
			{
				this.AttributeChanged("chk_ref", value);
			}
		}

		// Token: 0x1700014D RID: 333
		// (get) Token: 0x060004C2 RID: 1218 RVA: 0x000150C0 File Offset: 0x000132C0
		// (set) Token: 0x060004C3 RID: 1219 RVA: 0x000150CD File Offset: 0x000132CD
		public string Table
		{
			get
			{
				return base.GetAttribute("table");
			}
			set
			{
				this.AttributeChanged("table", value);
			}
		}

		// Token: 0x1700014E RID: 334
		// (get) Token: 0x060004C4 RID: 1220 RVA: 0x000150DB File Offset: 0x000132DB
		// (set) Token: 0x060004C5 RID: 1221 RVA: 0x000150E8 File Offset: 0x000132E8
		public string Column
		{
			get
			{
				return base.GetAttribute("column");
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				if (!string.IsNullOrEmpty(this.Table) && string.IsNullOrEmpty(value))
				{
					throw new Exception("'column' field is required");
				}
				this.AttributeChanged("column", value);
			}
		}

		// Token: 0x1700014F RID: 335
		// (get) Token: 0x060004C6 RID: 1222 RVA: 0x0001511F File Offset: 0x0001331F
		// (set) Token: 0x060004C7 RID: 1223 RVA: 0x0001512C File Offset: 0x0001332C
		public string Attribute
		{
			get
			{
				return base.GetAttribute("attribute");
			}
			set
			{
				this.AttributeChanged("attribute", value);
			}
		}

		// Token: 0x17000150 RID: 336
		// (get) Token: 0x060004C8 RID: 1224 RVA: 0x0001513A File Offset: 0x0001333A
		// (set) Token: 0x060004C9 RID: 1225 RVA: 0x00015147 File Offset: 0x00013347
		public string CanEdit
		{
			get
			{
				return base.GetAttribute("can_edit");
			}
			set
			{
				this.AttributeChanged("can_edit", value);
			}
		}

		// Token: 0x17000151 RID: 337
		// (get) Token: 0x060004CA RID: 1226 RVA: 0x00015155 File Offset: 0x00013355
		// (set) Token: 0x060004CB RID: 1227 RVA: 0x00015162 File Offset: 0x00013362
		public string CanQuery
		{
			get
			{
				return base.GetAttribute("can_query");
			}
			set
			{
				this.AttributeChanged("can_query", value);
			}
		}

		// Token: 0x17000152 RID: 338
		// (get) Token: 0x060004CC RID: 1228 RVA: 0x00015170 File Offset: 0x00013370
		// (set) Token: 0x060004CD RID: 1229 RVA: 0x0001517D File Offset: 0x0001337D
		public override string Widget
		{
			get
			{
				return base.GetAttribute("widget");
			}
			set
			{
				base.Widget = value;
			}
		}

		// Token: 0x060004CE RID: 1230 RVA: 0x00015188 File Offset: 0x00013388
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
			SpecAttributeUndoRedoCommand specAttributeUndoRedoCommand;
			if (key != null)
			{
				if (!(key == "table"))
				{
					if (key == "column")
					{
						specAttributeUndoRedoCommand = new SpecAttributeUndoRedoCommand(this);
						XElement columnInfo = TableColumnHelper.GetColumnInfo(this.Table, newValue);
						XElement columnAttrInfo = TableColumnHelper.GetColumnAttrInfo(this.Table, newValue);
						if (newValue == "")
						{
							string column = this.Column;
							string attribute2 = this.Attribute;
							specAttributeUndoRedoCommand.AddAttributeChanged("column", column, "");
							specAttributeUndoRedoCommand.AddAttributeChanged("attribute", attribute2, "");
						}
						if (columnInfo != null && !string.IsNullOrEmpty(newValue))
						{
							string column2 = this.Column;
							string attribute3 = this.Attribute;
							string value = columnInfo.Attribute("attribute").Value;
							string attribute4 = base.GetAttribute("type");
							string value2 = columnInfo.Attribute("type").Value;
							string req = this.Req;
							string value3 = columnInfo.Attribute("req").Value;
							if (columnAttrInfo != null)
							{
								specAttributeUndoRedoCommand.AddAttributeChanged("column", column2, newValue);
								specAttributeUndoRedoCommand.AddAttributeChanged("attribute", attribute3, value);
								string izoom = this.IZoom;
								string value4 = columnAttrInfo.Attribute("i_zoom").Value;
								string czoom = this.CZoom;
								string value5 = columnAttrInfo.Attribute("c_zoom").Value;
								string @default = this.Default;
								string value6 = columnAttrInfo.Attribute("default").Value;
								string max = this.Max;
								string value7 = columnAttrInfo.Attribute("max").Value;
								string min = this.Min;
								string value8 = columnAttrInfo.Attribute("min").Value;
								string chkRef = this.ChkRef;
								string value9 = columnAttrInfo.Attribute("chk_ref").Value;
								string items = this.Items;
								string value10 = columnAttrInfo.Attribute("items").Value;
								specAttributeUndoRedoCommand.AddAttributeChanged("i_zoom", izoom, value4);
								specAttributeUndoRedoCommand.AddAttributeChanged("c_zoom", czoom, value5);
								specAttributeUndoRedoCommand.AddAttributeChanged("default", @default, value6);
								specAttributeUndoRedoCommand.AddAttributeChanged("max", max, value7);
								specAttributeUndoRedoCommand.AddAttributeChanged("min", min, value8);
								specAttributeUndoRedoCommand.AddAttributeChanged("chk_ref", chkRef, value9);
								specAttributeUndoRedoCommand.AddAttributeChanged("items", items, value10);
							}
							specAttributeUndoRedoCommand.AddAttributeChanged("type", attribute4, value2);
							specAttributeUndoRedoCommand.AddAttributeChanged("req", req, value3);
							SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo;
							FormSpecModel formSpecModel = specificationInfo.FindNodeByName(this.Name);
							formSpecModel.GeneroComponent.SetAttribute("comment", "cmt_" + newValue);
							if (formSpecModel.GeneroComponent.BindElement != null)
							{
								XmlElement bindElement = formSpecModel.GeneroComponent.BindElement;
								string text = string.Format("lbl_{0}", newValue);
								string text2 = string.Format("cmt_{0}", newValue);
								string columnTextByFullName = TableColumnHelper.GetColumnTextByFullName(this.Table + "." + newValue);
								specificationInfo.SetFieldLocalStringText(text, columnTextByFullName);
								specificationInfo.SetFieldLocalStringText(text2, columnTextByFullName);
								specificationInfo.Rename(bindElement.Name, ComponentFactory.GetNewName(string.Format("lbl_{0}", newValue), base.ProgramKey));
								bindElement.SetAttribute("text", text);
								bindElement.SetAttribute("title", text);
								bindElement.SetAttribute("comment", text2);
								bindElement.OnPropertyChanged("Comment");
								bindElement.OnPropertyChanged("LocalString");
							}
							string name = this.Name;
							if (specificationInfo.IsExists(this.Table + "." + newValue))
							{
								this.Name = ComponentFactory.GetNewName(newValue, base.ProgramKey);
							}
							else
							{
								this.Name = this.Table + "." + newValue;
							}
							specAttributeUndoRedoCommand.AddAttributeChanged("Name", name, this.Name);
							goto IL_0469;
						}
						goto IL_0469;
					}
				}
				else
				{
					specAttributeUndoRedoCommand = new SpecAttributeUndoRedoCommand(this);
					specAttributeUndoRedoCommand.AddAttributeChanged(key, attribute, newValue);
					if (string.IsNullOrEmpty(newValue))
					{
						specAttributeUndoRedoCommand.AddAttributeChanged("column", this.Column, "");
						goto IL_0469;
					}
					goto IL_0469;
				}
			}
			specAttributeUndoRedoCommand = new SpecAttributeUndoRedoCommand(this);
			specAttributeUndoRedoCommand.AddAttributeChanged(key, attribute, newValue);
			IL_0469:
			specAttributeUndoRedoCommand.Execute();
			base.AttributeChanged(key, newValue);
			SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.DatabaseSource.RefreshUsed(SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.FormSpeDictionary);
		}

		// Token: 0x17000153 RID: 339
		// (get) Token: 0x060004CF RID: 1231 RVA: 0x000156A0 File Offset: 0x000138A0
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
				return (from e in citeSTD.Descendants("field")
					where e.Attribute("name").Value == this.Name && e.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select e).FirstOrDefault<XElement>();
			}
		}

		// Token: 0x17000154 RID: 340
		// (get) Token: 0x060004D0 RID: 1232 RVA: 0x0001570E File Offset: 0x0001390E
		public string Error
		{
			get
			{
				return string.Empty;
			}
		}

		// Token: 0x17000155 RID: 341
		public string this[string key]
		{
			get
			{
				string text = null;
				string attribute = base.GetAttribute("name");
				string attribute2 = base.GetAttribute("table");
				string attribute3 = base.GetAttribute("column");
				if (key != null)
				{
					if (!(key == "Name"))
					{
						if (key == "Column")
						{
							if (!string.IsNullOrEmpty(attribute2) && string.IsNullOrEmpty(attribute3))
							{
								text = "'column' field is required";
							}
						}
					}
					else if (string.IsNullOrEmpty(base.GetAttribute("name")))
					{
						text = "'name' field is required";
					}
					else if ((this.Table == "" || this.Column == "") && base.GetAttribute("name").IndexOf('.') != -1)
					{
						text = "Invalid value for 'name' property";
					}
					else if (attribute.Contains('.') && !string.Equals(string.Format("{0}.{1}", attribute2, attribute3), attribute))
					{
						text = "name format not match table/column setting";
					}
				}
				return text;
			}
		}

		// Token: 0x060004D2 RID: 1234 RVA: 0x00015810 File Offset: 0x00013A10
		public static SpecFieldNode Create(SpecificationInfo info, string name)
		{
			XElement xelement;
			if (info.GetCodeTemplate().Equals("p", StringComparison.CurrentCultureIgnoreCase) || info.GetCodeTemplate().Equals("r", StringComparison.CurrentCultureIgnoreCase))
			{
				xelement = new XElement("field", new object[]
				{
					new XAttribute("src", info.Env),
					new XAttribute("ver", info.Ver),
					new XAttribute("column", ""),
					new XAttribute("name", name),
					new XAttribute("table", ""),
					new XAttribute("attribute", ""),
					new XAttribute("type", ""),
					new XAttribute("req", ""),
					new XAttribute("i_zoom", ""),
					new XAttribute("c_zoom", ""),
					new XAttribute("chk_ref", ""),
					new XAttribute("items", ""),
					new XAttribute("default", ""),
					new XAttribute("max", ""),
					new XAttribute("min", ""),
					new XAttribute("can_edit", "N"),
					new XAttribute("can_query", "Y"),
					new XAttribute("widget", ""),
					new XAttribute("cite_std", "N"),
					new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE))
				});
			}
			else
			{
				xelement = new XElement("field", new object[]
				{
					new XAttribute("src", info.Env),
					new XAttribute("ver", info.Ver),
					new XAttribute("column", ""),
					new XAttribute("name", name),
					new XAttribute("table", ""),
					new XAttribute("attribute", ""),
					new XAttribute("type", ""),
					new XAttribute("req", ""),
					new XAttribute("i_zoom", ""),
					new XAttribute("c_zoom", ""),
					new XAttribute("chk_ref", ""),
					new XAttribute("items", ""),
					new XAttribute("default", ""),
					new XAttribute("max", ""),
					new XAttribute("min", ""),
					new XAttribute("can_edit", "Y"),
					new XAttribute("can_query", "Y"),
					new XAttribute("widget", ""),
					new XAttribute("cite_std", "N"),
					new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE))
				});
			}
			return new SpecFieldNode(info.Key, xelement);
		}

		// Token: 0x060004D3 RID: 1235 RVA: 0x00015C40 File Offset: 0x00013E40
		public static void SetDefaultProperties(PackageKey key, string widgetType, SpecFieldNode spec)
		{
			if (widgetType != null)
			{
				if (!(widgetType == "ComboBox") && !(widgetType == "RadioGroup") && !(widgetType == "CheckBox"))
				{
					return;
				}
				spec.SetAttribute("req", "Y");
			}
		}

		// Token: 0x060004D4 RID: 1236 RVA: 0x00015C8A File Offset: 0x00013E8A
		public static SpecFieldNode Create(PackageKey key, XElement source)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecFieldNode(key, source);
		}

		// Token: 0x060004D5 RID: 1237 RVA: 0x00015C98 File Offset: 0x00013E98
		public static SpecFieldNode Create(PackageKey key, XElement source, string src)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecFieldNode(key, source, src);
		}

		// Token: 0x17000156 RID: 342
		// (get) Token: 0x060004D6 RID: 1238 RVA: 0x00015CA8 File Offset: 0x00013EA8
		public bool IsReadOnly
		{
			get
			{
				FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.FindNodeByName(base.GetAttribute("name"));
				return formSpecModel != null && (formSpecModel.GeneroComponent.HasElementBinding || formSpecModel.GeneroComponent.IsCantDel);
			}
		}

		// Token: 0x17000157 RID: 343
		// (get) Token: 0x060004D7 RID: 1239 RVA: 0x00015D14 File Offset: 0x00013F14
		public IEnumerable<string> TableList
		{
			get
			{
				IEnumerable<string> enumerable = new List<string> { "" }.AsEnumerable<string>();
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo;
				IEnumerable<string> enumerable2 = from x in TableColumnHelper.GetOrderedTables(specificationInfo)
					select x.Attribute("name").Value;
				return enumerable.Concat<string>(enumerable2);
			}
		}

		// Token: 0x17000158 RID: 344
		// (get) Token: 0x060004D8 RID: 1240 RVA: 0x00015D98 File Offset: 0x00013F98
		public IEnumerable<string> TableColumns
		{
			get
			{
				IEnumerable<string> enumerable = new List<string> { "" }.AsEnumerable<string>();
				if (string.IsNullOrEmpty(this.Table))
				{
					return enumerable;
				}
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo;
				IEnumerable<XElement> enumerable2 = TableColumnHelper.FindTableColumns(this.Table, specificationInfo.Env);
				if (enumerable2 == null)
				{
					return enumerable;
				}
				return enumerable.Concat<string>(enumerable2.Select<XElement, string>((XElement x) => x.Attribute("name").Value));
			}
		}
	}
}
