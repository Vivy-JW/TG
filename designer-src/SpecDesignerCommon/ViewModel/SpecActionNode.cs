using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.UndoRedoCommands;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x0200003E RID: 62
	public class SpecActionNode : AbstractSpecNode, IDataErrorInfo
	{
		// Token: 0x060001FE RID: 510 RVA: 0x00008E28 File Offset: 0x00007028
		public SpecActionNode(PackageKey key, XElement source)
			: base(key, source)
		{
			if ((this.Status & SpecStatus.DELETE) == SpecStatus.DELETE || (this.Status & SpecStatus.CREATE) == SpecStatus.CREATE)
			{
				this._isDisabled = true;
			}
		}

		// Token: 0x060001FF RID: 511 RVA: 0x00008E67 File Offset: 0x00007067
		public SpecActionNode(PackageKey key, XElement source, string src)
			: base(key, source, src)
		{
			if ((this.Status & SpecStatus.DELETE) == SpecStatus.DELETE || (this.Status & SpecStatus.CREATE) == SpecStatus.CREATE)
			{
				this._isDisabled = true;
			}
		}

		// Token: 0x17000079 RID: 121
		// (get) Token: 0x06000200 RID: 512 RVA: 0x00008EA7 File Offset: 0x000070A7
		// (set) Token: 0x06000201 RID: 513 RVA: 0x00008EB4 File Offset: 0x000070B4
		public override string Name
		{
			get
			{
				return base.GetAttribute("id");
			}
			set
			{
				if (base.IsCited || value == this.Name)
				{
					return;
				}
				RenameUndoRedoCommand renameUndoRedoCommand = new RenameUndoRedoCommand(base.ProgramKey, this.Name, value);
				renameUndoRedoCommand.Execute();
				if (SettingManager.Get().GetUndoRedoManager(base.ProgramKey) != null)
				{
					SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddUndo(renameUndoRedoCommand);
				}
			}
		}

		// Token: 0x06000202 RID: 514 RVA: 0x00008F19 File Offset: 0x00007119
		internal override void SetName(FormSpecModel model, string newName)
		{
			base.SetAttribute("id", newName);
			this.Status |= SpecStatus.MODIFY;
			this.OnPropertyChanged("Name");
		}

		// Token: 0x1700007A RID: 122
		// (get) Token: 0x06000203 RID: 515 RVA: 0x00008F40 File Offset: 0x00007140
		// (set) Token: 0x06000204 RID: 516 RVA: 0x00008F4D File Offset: 0x0000714D
		public string Gencode
		{
			get
			{
				return base.GetAttribute("gen_code");
			}
			set
			{
				if (string.IsNullOrEmpty(value))
				{
					return;
				}
				this.AttributeChanged("gen_code", value);
			}
		}

		// Token: 0x1700007B RID: 123
		// (get) Token: 0x06000205 RID: 517 RVA: 0x00008F64 File Offset: 0x00007164
		// (set) Token: 0x06000206 RID: 518 RVA: 0x00008F71 File Offset: 0x00007171
		public string Type
		{
			get
			{
				return base.GetAttribute("type");
			}
			set
			{
				if (string.IsNullOrEmpty(value))
				{
					return;
				}
				this.AttributeChanged("type", value);
			}
		}

		// Token: 0x1700007C RID: 124
		// (get) Token: 0x06000207 RID: 519 RVA: 0x00008F88 File Offset: 0x00007188
		// (set) Token: 0x06000208 RID: 520 RVA: 0x00008FE8 File Offset: 0x000071E8
		public string LocalString
		{
			get
			{
				string text = string.Empty;
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo;
				FormSpecModel formSpecModel = specificationInfo.FindNodeByName(this.Name);
				if (formSpecModel != null)
				{
					text = formSpecModel.GeneroComponent.LocalString;
				}
				else
				{
					text = specificationInfo.GetActLocalStringText(this.Name);
				}
				if (text != null)
				{
					return text;
				}
				return this.Name;
			}
			set
			{
				ActLocalStringUndoRedoCommand actLocalStringUndoRedoCommand = new ActLocalStringUndoRedoCommand(this, value);
				actLocalStringUndoRedoCommand.Execute();
				if (SettingManager.Get().GetUndoRedoManager(base.ProgramKey) != null)
				{
					SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddUndo(actLocalStringUndoRedoCommand);
				}
			}
		}

		// Token: 0x1700007D RID: 125
		// (get) Token: 0x06000209 RID: 521 RVA: 0x0000902B File Offset: 0x0000722B
		// (set) Token: 0x0600020A RID: 522 RVA: 0x00009033 File Offset: 0x00007233
		public bool IsDisabled
		{
			get
			{
				return this._isDisabled;
			}
			set
			{
				this._isDisabled = value;
				this.OnPropertyChanged("IsDisabled");
				this.Status = (this._isDisabled ? SpecStatus.DELETE : SpecStatus.MODIFY);
			}
		}

		// Token: 0x0600020B RID: 523 RVA: 0x0000905C File Offset: 0x0000725C
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
			specAttributeUndoRedoCommand.AddAttributeChanged(key, base.GetAttribute(key), newValue);
			specAttributeUndoRedoCommand.Execute();
			base.AttributeChanged(key, newValue);
		}

		// Token: 0x1700007E RID: 126
		// (get) Token: 0x0600020C RID: 524 RVA: 0x00009100 File Offset: 0x00007300
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
				return (from e in citeSTD.Descendants("act")
					where e.Attribute("id").Value == this.Name && e.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select e).FirstOrDefault<XElement>();
			}
		}

		// Token: 0x1700007F RID: 127
		// (get) Token: 0x0600020D RID: 525 RVA: 0x0000916E File Offset: 0x0000736E
		// (set) Token: 0x0600020E RID: 526 RVA: 0x0000917B File Offset: 0x0000737B
		public string ActionTypes
		{
			get
			{
				return base.GetAttribute("type");
			}
			set
			{
				this.AttributeChanged("type", value);
			}
		}

		// Token: 0x17000080 RID: 128
		// (get) Token: 0x0600020F RID: 527 RVA: 0x0000918C File Offset: 0x0000738C
		public bool IsToolBarAction
		{
			get
			{
				if (this._isToolBarAction == null)
				{
					SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo;
					if (specificationInfo == null)
					{
						return false;
					}
					if (specificationInfo.AllowedAction == null)
					{
						return false;
					}
					this._isToolBarAction = new bool?(specificationInfo.AllowedAction.Contains(this.Name));
				}
				return this._isToolBarAction == true;
			}
		}

		// Token: 0x17000081 RID: 129
		// (get) Token: 0x06000210 RID: 528 RVA: 0x0000923C File Offset: 0x0000743C
		public bool IsActionDefaults
		{
			get
			{
				if (this._isActionDefaults == null)
				{
					XElement actionDefaults = SettingManager.Get().GetTzpManger(base.ProgramKey).ActionDefaults;
					if (actionDefaults == null)
					{
						return false;
					}
					int num = (from a in actionDefaults.Elements("ActionDefault")
						where a.Attribute("name") != null && a.Attribute("name").Value == this.Name
						select a).Count<XElement>();
					this._isActionDefaults = new bool?(num > 0);
				}
				return this._isActionDefaults == true;
			}
		}

		// Token: 0x06000211 RID: 529 RVA: 0x000092D0 File Offset: 0x000074D0
		public bool IsContainsType(string type)
		{
			if (this.ActionTypes == null)
			{
				return false;
			}
			string[] array = this.ActionTypes.Split(new char[] { ',' });
			foreach (string text in array)
			{
				if (Regex.IsMatch(text, type))
				{
					return true;
				}
			}
			return false;
		}

		// Token: 0x06000212 RID: 530 RVA: 0x0000932C File Offset: 0x0000752C
		public void AddActionType(string type)
		{
			if (string.IsNullOrEmpty(this.ActionTypes))
			{
				throw new Exception(Application.Current.FindResource("Message_ActionHasType") as string);
			}
			if (this.IsContainsType(type))
			{
				return;
			}
			List<string> list = this.ActionTypes.Split(new char[] { ',' }).ToList<string>();
			list.Add(type);
			if (type == "all")
			{
				List<string> list2 = new List<string>(list);
				using (List<string>.Enumerator enumerator = list2.GetEnumerator())
				{
					while (enumerator.MoveNext())
					{
						string text = enumerator.Current;
						if (text.StartsWith("db"))
						{
							list.Remove(text);
						}
					}
					goto IL_00C3;
				}
			}
			if (type.StartsWith("db"))
			{
				list.Remove("all");
			}
			IL_00C3:
			this.ActionTypes = string.Join(",", list);
		}

		// Token: 0x06000213 RID: 531 RVA: 0x00009420 File Offset: 0x00007620
		public void RemoveActionType(string type)
		{
			if (string.IsNullOrEmpty(this.ActionTypes))
			{
				throw new Exception(Application.Current.FindResource("Message_ActionHasType") as string);
			}
			if (!this.IsContainsType(type))
			{
				return;
			}
			List<string> list = this.ActionTypes.Split(new char[] { ',' }).ToList<string>();
			list.Remove(type);
			if (list.Count<string>() == 0)
			{
				list.Add("all");
			}
			this.ActionTypes = string.Join(",", list);
		}

		// Token: 0x06000214 RID: 532 RVA: 0x000094A8 File Offset: 0x000076A8
		public static SpecActionNode Create(SpecificationInfo info, string name)
		{
			XElement xelement = new XElement("act", new object[]
			{
				new XAttribute("src", info.Env),
				new XAttribute("ver", info.Ver),
				new XAttribute("id", name),
				new XAttribute("cite_std", "N"),
				new XAttribute("gen_code", "Y"),
				new XAttribute("type", "all"),
				new XAttribute("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE))
			});
			return new SpecActionNode(info.Key, xelement);
		}

		// Token: 0x06000215 RID: 533 RVA: 0x0000957E File Offset: 0x0000777E
		public static SpecActionNode Create(PackageKey key, XElement source)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecActionNode(key, source);
		}

		// Token: 0x06000216 RID: 534 RVA: 0x0000958C File Offset: 0x0000778C
		public static SpecActionNode Create(PackageKey key, XElement source, string src)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecActionNode(key, source, src);
		}

		// Token: 0x06000217 RID: 535 RVA: 0x0000959B File Offset: 0x0000779B
		public static SpecActionNode Create(SpecificationInfo info)
		{
			return SpecActionNode.Create(info, info.GetNewActionID());
		}

		// Token: 0x17000082 RID: 130
		// (get) Token: 0x06000218 RID: 536 RVA: 0x000095A9 File Offset: 0x000077A9
		public string Error
		{
			get
			{
				return string.Empty;
			}
		}

		// Token: 0x17000083 RID: 131
		public string this[string columnName]
		{
			get
			{
				string text = null;
				if (columnName != null && columnName == "ID" && string.IsNullOrEmpty(base.GetAttribute("id")))
				{
					text = "'id' field is required";
				}
				return text;
			}
		}

		// Token: 0x040000BD RID: 189
		private bool _isDisabled;

		// Token: 0x040000BE RID: 190
		private bool? _isToolBarAction = null;

		// Token: 0x040000BF RID: 191
		private bool? _isActionDefaults = null;
	}
}
