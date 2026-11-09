using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Input;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;
using SpecDesignerCommon.UndoRedoCommands;
using SpecDesignerPreference;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000060 RID: 96
	public class XmlElement : INotifyPropertyChanged, ISpecSearchable
	{
		// Token: 0x170000C2 RID: 194
		// (get) Token: 0x0600032D RID: 813 RVA: 0x0000CE9A File Offset: 0x0000B09A
		// (set) Token: 0x0600032E RID: 814 RVA: 0x0000CEA2 File Offset: 0x0000B0A2
		public PackageKey Key { get; private set; }

		// Token: 0x170000C3 RID: 195
		// (get) Token: 0x0600032F RID: 815 RVA: 0x0000CEAB File Offset: 0x0000B0AB
		// (set) Token: 0x06000330 RID: 816 RVA: 0x0000CEB4 File Offset: 0x0000B0B4
		public XmlElement Parent
		{
			get
			{
				return this._parent;
			}
			set
			{
				this._parent = value;
				if (this.Parent != null)
				{
					if (this.NodeName == "DateTimeEdit" && SettingManager.Get().ErpVer == "1.0")
					{
						return;
					}
					ComponentFactory.AttachDefaultAttributes(this.Key, this, false);
				}
				this.OnPropertyChanged("Parent");
			}
		}

		// Token: 0x170000C4 RID: 196
		// (get) Token: 0x06000331 RID: 817 RVA: 0x0000CF12 File Offset: 0x0000B112
		public string NodeName
		{
			get
			{
				return this._nodeName;
			}
		}

		// Token: 0x170000C5 RID: 197
		// (get) Token: 0x06000332 RID: 818 RVA: 0x0000CF1A File Offset: 0x0000B11A
		// (set) Token: 0x06000333 RID: 819 RVA: 0x0000CF22 File Offset: 0x0000B122
		private XElement Source { get; set; }

		// Token: 0x170000C6 RID: 198
		// (get) Token: 0x06000334 RID: 820 RVA: 0x0000CF2B File Offset: 0x0000B12B
		public bool IsCantDel
		{
			get
			{
				return SettingManager.Get().GetTzpManger(this.Key).IsNormalStyle && this.HasCantDelTags(this.GetAttribute("tag"));
			}
		}

		// Token: 0x170000C7 RID: 199
		// (get) Token: 0x06000335 RID: 821 RVA: 0x0000CF58 File Offset: 0x0000B158
		public bool IsCantMove
		{
			get
			{
				if ("patch_grid" == this.Name)
				{
					return true;
				}
				if (ComponentType.Form == this.Type)
				{
					return true;
				}
				if (this.GetAttribute("tag") == null)
				{
					return false;
				}
				List<string> list = this.GetAttribute("tag").ToLower().Replace(" ", "")
					.Split(new char[] { ',' })
					.ToList<string>();
				if (list.Contains("cantmove"))
				{
					return true;
				}
				foreach (XmlElement xmlElement in this.Nodes)
				{
					if (xmlElement.IsCantMove)
					{
						return true;
					}
				}
				return false;
			}
		}

		// Token: 0x170000C8 RID: 200
		// (get) Token: 0x06000336 RID: 822 RVA: 0x0000D028 File Offset: 0x0000B228
		// (set) Token: 0x06000337 RID: 823 RVA: 0x0000D078 File Offset: 0x0000B278
		public bool IsRequired
		{
			get
			{
				string attribute = this.GetAttribute("style");
				return !string.IsNullOrEmpty(attribute) && attribute.ToLower().Split(new char[] { ' ' }).ToList<string>()
					.Contains("required");
			}
			set
			{
				ComponentType type = this.Type;
				if (type == ComponentType.RadioGroup || type == ComponentType.CheckBox)
				{
					return;
				}
				string attribute = this.GetAttribute("style");
				List<string> list = attribute.Split(new char[] { ' ' }).Distinct<string>().ToList<string>();
				switch (value)
				{
				case false:
					this["required"] = "false";
					this["notNull"] = "false";
					list.Remove("required");
					this["unhidable"] = "false";
					break;
				case true:
					this["required"] = "true";
					this["notNull"] = "true";
					if (list.Contains(XmlElement.NOSET))
					{
						list.Remove(XmlElement.NOSET);
					}
					if (!list.Contains("required"))
					{
						list.Add("required");
					}
					if (this.Parent != null && this.Parent.Type == ComponentType.Table)
					{
						this["unhidable"] = "true";
					}
					break;
				}
				this.SetAttribute("style", string.Join(" ", list.ToArray()).Trim());
				this.OnPropertyChanged("IsRequired");
			}
		}

		// Token: 0x170000C9 RID: 201
		// (get) Token: 0x06000338 RID: 824 RVA: 0x0000D1B8 File Offset: 0x0000B3B8
		// (set) Token: 0x06000339 RID: 825 RVA: 0x0000D1D8 File Offset: 0x0000B3D8
		public bool IsHidden
		{
			get
			{
				bool.TryParse(this["hidden"], out this.isHidden);
				return this.isHidden;
			}
			set
			{
				if (SettingManager.Get().GetTzpManger(this.Key).IsSimpleRefFormHidden(this.Name))
				{
					this.isHidden = value;
					this["hidden"] = value.ToString().ToLower();
					this.OnPropertyChanged("IsHidden");
					return;
				}
				if (!this.IsBatchSet)
				{
					string text = Application.Current.FindResource("SimpleForm_HintCantShow") as string;
					DesignerMessageBox.Show(text);
				}
			}
		}

		// Token: 0x170000CA RID: 202
		// (get) Token: 0x0600033A RID: 826 RVA: 0x0000D250 File Offset: 0x0000B450
		// (set) Token: 0x0600033B RID: 827 RVA: 0x0000D258 File Offset: 0x0000B458
		public bool IsBatchSet { get; set; }

		// Token: 0x0600033C RID: 828 RVA: 0x0000D264 File Offset: 0x0000B464
		private static void GetHiddenRecursive(List<XmlElement> list, XmlElement xe)
		{
			if (xe != null && list != null)
			{
				list.Add(xe);
				foreach (XmlElement xmlElement in xe.Nodes)
				{
					XmlElement.GetHiddenRecursive(list, xmlElement);
				}
			}
		}

		// Token: 0x0600033D RID: 829 RVA: 0x0000D2C0 File Offset: 0x0000B4C0
		public void BatchSetHidden(bool val)
		{
			List<XmlElement> list = new List<XmlElement>();
			XmlElement.GetHiddenRecursive(list, this);
			MultiHideUndoRedoCommand multiHideUndoRedoCommand = new MultiHideUndoRedoCommand(list, val);
			SettingManager.Get().GetUndoRedoManager(this.Key).AddThenExecute(multiHideUndoRedoCommand);
		}

		// Token: 0x170000CB RID: 203
		// (get) Token: 0x0600033E RID: 830 RVA: 0x0000D2F8 File Offset: 0x0000B4F8
		public bool IsPKField
		{
			get
			{
				return TableColumnHelper.IsPK(this.GetAttribute("sqlTabName"), this.GetAttribute("colName"));
			}
		}

		// Token: 0x170000CC RID: 204
		// (get) Token: 0x0600033F RID: 831 RVA: 0x0000D318 File Offset: 0x0000B518
		public bool IsSpecificationDefined
		{
			get
			{
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo;
				FormSpecModel formSpecModel = specificationInfo.FindNodeByName(this.Name);
				return formSpecModel != null && formSpecModel.SpecNode != null && !string.IsNullOrEmpty(formSpecModel.SpecNode.CDATA);
			}
		}

		// Token: 0x170000CD RID: 205
		// (get) Token: 0x06000340 RID: 832 RVA: 0x0000D36C File Offset: 0x0000B56C
		public bool IsUnCited
		{
			get
			{
				if (SettingManager.Get().GetTzpManger(this.Key).IsStandardProgram)
				{
					return false;
				}
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo;
				FormSpecModel formSpecModel = specificationInfo.FindNodeByName(this.Name);
				return formSpecModel != null && !formSpecModel.IsCited;
			}
		}

		// Token: 0x170000CE RID: 206
		// (get) Token: 0x06000341 RID: 833 RVA: 0x0000D3C4 File Offset: 0x0000B5C4
		public SpecStatus SpecNodeStatus
		{
			get
			{
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo;
				FormSpecModel formSpecModel = specificationInfo.FindNodeByName(this.Name);
				if (formSpecModel != null)
				{
					return formSpecModel.SpecNodeStatus;
				}
				return SpecStatus.NULL;
			}
		}

		// Token: 0x170000CF RID: 207
		// (get) Token: 0x06000342 RID: 834 RVA: 0x0000D400 File Offset: 0x0000B600
		// (set) Token: 0x06000343 RID: 835 RVA: 0x0000D423 File Offset: 0x0000B623
		public int TabIndex
		{
			get
			{
				int num = -1;
				int.TryParse(this.GetAttribute("tabIndex"), out num);
				return num;
			}
			set
			{
				int tabIndex = this.TabIndex;
				if (value == this.TabIndex)
				{
					return;
				}
				this.SetAttribute("tabIndex", value.ToString());
				this.OnPropertyChanged("TabIndex");
			}
		}

		// Token: 0x06000344 RID: 836 RVA: 0x0000D454 File Offset: 0x0000B654
		public static void GetMaxTabIndex(XmlElement xe, ref int maxTabIndex)
		{
			if (maxTabIndex < xe.TabIndex)
			{
				maxTabIndex = xe.TabIndex;
			}
			foreach (XmlElement xmlElement in xe.Nodes)
			{
				XmlElement.GetMaxTabIndex(xmlElement, ref maxTabIndex);
			}
		}

		// Token: 0x170000D0 RID: 208
		// (get) Token: 0x06000345 RID: 837 RVA: 0x0000D4B4 File Offset: 0x0000B6B4
		// (set) Token: 0x06000346 RID: 838 RVA: 0x0000D51C File Offset: 0x0000B71C
		public string Text
		{
			get
			{
				ComponentType type = this.Type;
				if (type == ComponentType.Item)
				{
					return this.GetAttribute("text");
				}
				string attribute = this.GetAttribute("text");
				string text = (string.IsNullOrEmpty(attribute) ? attribute : SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.GetFieldLocalStringText(attribute));
				if (!string.IsNullOrEmpty(text))
				{
					return text;
				}
				return this.Name;
			}
			set
			{
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo;
				ComponentType type = this.Type;
				if (type == ComponentType.Item)
				{
					if (value == this.Text)
					{
						return;
					}
					this.SetAttribute("text", value);
				}
				else
				{
					string attribute = this.GetAttribute("text");
					string fieldLocalStringText = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.GetFieldLocalStringText(attribute);
					if (value == fieldLocalStringText)
					{
						return;
					}
					specificationInfo.SetFieldLocalStringText(attribute, value);
				}
				this.OnPropertyChanged("Text");
			}
		}

		// Token: 0x170000D1 RID: 209
		// (get) Token: 0x06000347 RID: 839 RVA: 0x0000D5B0 File Offset: 0x0000B7B0
		// (set) Token: 0x06000348 RID: 840 RVA: 0x0000D620 File Offset: 0x0000B820
		public string Comment
		{
			get
			{
				string attribute = this.GetAttribute("comment");
				if (attribute == null)
				{
					return null;
				}
				if (this.NodeName.Equals(ComponentType.Button.ToString()))
				{
					return SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.GetActLocalStringText(attribute);
				}
				return SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.GetFieldLocalStringText(attribute);
			}
			set
			{
				string attribute = this.GetAttribute("comment");
				if (value == attribute)
				{
					return;
				}
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo;
				if (this.NodeName.Equals(ComponentType.Button.ToString()))
				{
					specificationInfo.SetActLocalStringText(attribute, value);
				}
				else
				{
					specificationInfo.SetFieldLocalStringText(attribute, value);
				}
				this.OnPropertyChanged("Comment");
			}
		}

		// Token: 0x170000D2 RID: 210
		// (get) Token: 0x06000349 RID: 841 RVA: 0x0000D68F File Offset: 0x0000B88F
		// (set) Token: 0x0600034A RID: 842 RVA: 0x0000D697 File Offset: 0x0000B897
		public bool IsShowTabIndex
		{
			get
			{
				return this._isShowTabIndex;
			}
			set
			{
				if (this._isShowTabIndex == value)
				{
					return;
				}
				this._isShowTabIndex = value;
				this.OnPropertyChanged("IsShowTabIndex");
			}
		}

		// Token: 0x170000D3 RID: 211
		// (get) Token: 0x0600034B RID: 843 RVA: 0x0000D6B5 File Offset: 0x0000B8B5
		// (set) Token: 0x0600034C RID: 844 RVA: 0x0000D6C0 File Offset: 0x0000B8C0
		public bool IsSelected
		{
			get
			{
				return this._isSelected;
			}
			set
			{
				if (this._isSelected == value)
				{
					return;
				}
				this._isSelected = value;
				this.OnPropertyChanged("IsSelected");
				if (this.Type == ComponentType.Page && this._isSelected)
				{
					this.IsFocused = this._isSelected;
				}
				if (this.Parent != null && this._isSelected)
				{
					this.Parent.IsFocused = (this.Parent.IsExpanded = this._isSelected);
				}
			}
		}

		// Token: 0x170000D4 RID: 212
		// (get) Token: 0x0600034D RID: 845 RVA: 0x0000D735 File Offset: 0x0000B935
		// (set) Token: 0x0600034E RID: 846 RVA: 0x0000D73D File Offset: 0x0000B93D
		public bool IsFocused
		{
			get
			{
				return this._isFocused;
			}
			set
			{
				this._isFocused = value;
				this.OnPropertyChanged("IsFocused");
				if (this.Parent != null)
				{
					this.Parent.IsFocused = this.IsFocused;
				}
			}
		}

		// Token: 0x170000D5 RID: 213
		// (get) Token: 0x0600034F RID: 847 RVA: 0x0000D76C File Offset: 0x0000B96C
		public int StepX
		{
			get
			{
				int num = 0;
				int.TryParse(this.GetAttribute("stepX"), out num);
				return num;
			}
		}

		// Token: 0x170000D6 RID: 214
		// (get) Token: 0x06000350 RID: 848 RVA: 0x0000D790 File Offset: 0x0000B990
		public int StepY
		{
			get
			{
				int num = 0;
				int.TryParse(this.GetAttribute("stepY"), out num);
				return num;
			}
		}

		// Token: 0x170000D7 RID: 215
		// (get) Token: 0x06000351 RID: 849 RVA: 0x0000D7B4 File Offset: 0x0000B9B4
		public int ColumnCount
		{
			get
			{
				int num = 0;
				int.TryParse(this.GetAttribute("columnCount"), out num);
				return num;
			}
		}

		// Token: 0x170000D8 RID: 216
		// (get) Token: 0x06000352 RID: 850 RVA: 0x0000D7D8 File Offset: 0x0000B9D8
		public int RowCount
		{
			get
			{
				int num = 0;
				int.TryParse(this.GetAttribute("rowCount"), out num);
				return num;
			}
		}

		// Token: 0x170000D9 RID: 217
		// (get) Token: 0x06000353 RID: 851 RVA: 0x0000D7FC File Offset: 0x0000B9FC
		public int RowHeight
		{
			get
			{
				switch (this.Type)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
				{
					int num = 1;
					if (this.GetAttribute("rowHeight") != null)
					{
						int.TryParse(this.GetAttribute("rowHeight"), out num);
					}
					return num;
				}
				default:
					return 0;
				}
			}
		}

		// Token: 0x170000DA RID: 218
		// (get) Token: 0x06000354 RID: 852 RVA: 0x0000D848 File Offset: 0x0000BA48
		public int TotalRows
		{
			get
			{
				switch (this.Type)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
				{
					int num = 1;
					if (this.GetAttribute("totalRows") != null)
					{
						int.TryParse(this.GetAttribute("totalRows"), out num);
					}
					return num;
				}
				default:
					return 0;
				}
			}
		}

		// Token: 0x170000DB RID: 219
		// (get) Token: 0x06000355 RID: 853 RVA: 0x0000D894 File Offset: 0x0000BA94
		public bool IsEnabledAggregate
		{
			get
			{
				if (this.Parent == null)
				{
					return false;
				}
				switch (this.Parent.Type)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
					return this.GetAttribute("aggregate") != null && this.GetAttribute("aggregate").Equals("true", StringComparison.InvariantCultureIgnoreCase);
				default:
					return false;
				}
			}
		}

		// Token: 0x170000DC RID: 220
		// (get) Token: 0x06000356 RID: 854 RVA: 0x0000D8F4 File Offset: 0x0000BAF4
		public bool IsAnySiblingEnabledAggregate
		{
			get
			{
				if (this.Parent == null)
				{
					return false;
				}
				bool flag = false;
				ComponentType type = this.Parent.Type;
				if (type == ComponentType.Table)
				{
					foreach (XmlElement xmlElement in this.Parent.Nodes)
					{
						if (xmlElement.GetAttribute("aggregate") != null && xmlElement.GetAttribute("aggregate").Equals("true", StringComparison.InvariantCultureIgnoreCase))
						{
							flag = true;
							break;
						}
					}
				}
				return flag;
			}
		}

		// Token: 0x170000DD RID: 221
		// (get) Token: 0x06000357 RID: 855 RVA: 0x0000D988 File Offset: 0x0000BB88
		// (set) Token: 0x06000358 RID: 856 RVA: 0x0000D996 File Offset: 0x0000BB96
		public int Width
		{
			get
			{
				return this.GridWidth * FormDesignSetting.UnitWidth;
			}
			set
			{
				this.GridWidth = FormDesignSetting.TransformToGridWidth((double)value);
			}
		}

		// Token: 0x170000DE RID: 222
		// (get) Token: 0x06000359 RID: 857 RVA: 0x0000D9A5 File Offset: 0x0000BBA5
		// (set) Token: 0x0600035A RID: 858 RVA: 0x0000D9B3 File Offset: 0x0000BBB3
		public int Height
		{
			get
			{
				return this.GridHeight * FormDesignSetting.UnitHeight;
			}
			set
			{
				this.GridHeight = FormDesignSetting.TransformToGridHeight((double)value);
			}
		}

		// Token: 0x170000DF RID: 223
		// (get) Token: 0x0600035B RID: 859 RVA: 0x0000D9C2 File Offset: 0x0000BBC2
		// (set) Token: 0x0600035C RID: 860 RVA: 0x0000D9D0 File Offset: 0x0000BBD0
		public int X
		{
			get
			{
				return this.GridX * FormDesignSetting.UnitWidth;
			}
			set
			{
				this.GridX = FormDesignSetting.TransformToGridWidth((double)value);
			}
		}

		// Token: 0x170000E0 RID: 224
		// (get) Token: 0x0600035D RID: 861 RVA: 0x0000D9DF File Offset: 0x0000BBDF
		// (set) Token: 0x0600035E RID: 862 RVA: 0x0000D9ED File Offset: 0x0000BBED
		public int Y
		{
			get
			{
				return this.GridY * FormDesignSetting.UnitHeight;
			}
			set
			{
				this.GridY = FormDesignSetting.TransformToGridHeight((double)value);
			}
		}

		// Token: 0x170000E1 RID: 225
		// (get) Token: 0x0600035F RID: 863 RVA: 0x0000D9FC File Offset: 0x0000BBFC
		// (set) Token: 0x06000360 RID: 864 RVA: 0x0000DA6C File Offset: 0x0000BC6C
		public int GridWidth
		{
			get
			{
				int num = 0;
				ComponentType type = this.Type;
				if (type != ComponentType.Page)
				{
					if (type == ComponentType.Phantom)
					{
						if (this.Parent != null && (this.Parent.Type == ComponentType.Tree || this.Parent.Type == ComponentType.Table))
						{
							num = 2;
						}
						else
						{
							num = 0;
						}
					}
					else
					{
						int.TryParse(this.GetAttribute("gridWidth"), out num);
					}
				}
				else
				{
					num = this.Parent.GridWidth;
				}
				return num;
			}
			set
			{
				if (this.GetAttribute("gridWidth") == null || value == this.GridWidth)
				{
					return;
				}
				int num = ((value > this.GridWidth) ? value : Math.Max(this.MinGridWidth, value));
				if (num == this.GridWidth)
				{
					return;
				}
				FormSizeUndoRedoCommand formSizeUndoRedoCommand = new FormSizeUndoRedoCommand(this, "gridWidth", num);
				formSizeUndoRedoCommand.Execute();
			}
		}

		// Token: 0x170000E2 RID: 226
		// (get) Token: 0x06000361 RID: 865 RVA: 0x0000DAC8 File Offset: 0x0000BCC8
		// (set) Token: 0x06000362 RID: 866 RVA: 0x0000DB30 File Offset: 0x0000BD30
		public int GridHeight
		{
			get
			{
				int num = 0;
				ComponentType type = this.Type;
				if (type == ComponentType.HLine)
				{
					num = 1;
				}
				else
				{
					if (this.Parent != null && (this.Parent.Type == ComponentType.Table || this.Parent.Type == ComponentType.Tree))
					{
						return this.Parent.RowHeight;
					}
					int.TryParse(this.GetAttribute("gridHeight"), out num);
				}
				return num;
			}
			set
			{
				if (this.GetAttribute("gridHeight") == null || value == this.GridHeight)
				{
					return;
				}
				int num = ((value > this.GridHeight) ? value : Math.Max(this.MinGridHeight, value));
				if (this.GridHeight == num)
				{
					return;
				}
				FormSizeUndoRedoCommand formSizeUndoRedoCommand = new FormSizeUndoRedoCommand(this, "gridHeight", num);
				formSizeUndoRedoCommand.Execute();
			}
		}

		// Token: 0x170000E3 RID: 227
		// (get) Token: 0x06000363 RID: 867 RVA: 0x0000DB8C File Offset: 0x0000BD8C
		// (set) Token: 0x06000364 RID: 868 RVA: 0x0000DBB0 File Offset: 0x0000BDB0
		public int GridX
		{
			get
			{
				int num = 0;
				int.TryParse(this.GetAttribute("posX"), out num);
				return num;
			}
			set
			{
				if (this.GetAttribute("posX") == null || value == this.GridX)
				{
					return;
				}
				int num = Math.Max(this.MinGridX, value);
				if (this.GridX == num)
				{
					return;
				}
				FormPosUndoRedoCommand formPosUndoRedoCommand = new FormPosUndoRedoCommand(this, "posX", num);
				formPosUndoRedoCommand.Execute();
			}
		}

		// Token: 0x06000365 RID: 869 RVA: 0x0000DBFE File Offset: 0x0000BDFE
		public void SetInitGridX()
		{
			if (SettingManager.Get().CheckUndoRedoManager(this.Key))
			{
				throw new Exception("illeagal call SetInitGridX");
			}
			this.SetAttribute("posX", "0");
		}

		// Token: 0x170000E4 RID: 228
		// (get) Token: 0x06000366 RID: 870 RVA: 0x0000DC30 File Offset: 0x0000BE30
		// (set) Token: 0x06000367 RID: 871 RVA: 0x0000DC54 File Offset: 0x0000BE54
		public int GridY
		{
			get
			{
				int num = 0;
				int.TryParse(this.GetAttribute("posY"), out num);
				return num;
			}
			set
			{
				if (this.GetAttribute("posY") == null || value == this.GridY)
				{
					return;
				}
				int num = Math.Max(this.MinGridY, value);
				if (this.GridY == num)
				{
					return;
				}
				FormPosUndoRedoCommand formPosUndoRedoCommand = new FormPosUndoRedoCommand(this, "posY", num);
				formPosUndoRedoCommand.Execute();
			}
		}

		// Token: 0x06000368 RID: 872 RVA: 0x0000DCA2 File Offset: 0x0000BEA2
		public void SetInitGridY()
		{
			if (SettingManager.Get().CheckUndoRedoManager(this.Key))
			{
				throw new Exception("illeagal call SetInitGridY");
			}
			this.SetAttribute("posY", "0");
		}

		// Token: 0x170000E5 RID: 229
		// (get) Token: 0x06000369 RID: 873 RVA: 0x0000DCD4 File Offset: 0x0000BED4
		public int MinGridWidth
		{
			get
			{
				int num = 1;
				if (this.GetAttribute("minWidth") != null)
				{
					int.TryParse(this.GetAttribute("minWidth"), out num);
				}
				switch (this.Type)
				{
				case ComponentType.Folder:
				case ComponentType.Grid:
				case ComponentType.Group:
				case ComponentType.VBox:
					num = Math.Max(3, num);
					return Math.Max(this.GetContentWidth(), num);
				case ComponentType.Form:
					num = Math.Max(2, num);
					return Math.Max(this.GetContentWidth(), num);
				case ComponentType.HBox:
					num = Math.Max(7, num);
					return Math.Max(this.GetContentWidth(), num);
				case ComponentType.RadioGroup:
				case ComponentType.Label:
				case ComponentType.Edit:
				case ComponentType.ProgressBar:
				case ComponentType.ComboBox:
				case ComponentType.TextEdit:
				case ComponentType.Button:
				case ComponentType.ButtonEdit:
				case ComponentType.DateEdit:
				case ComponentType.Canvas:
				case ComponentType.CheckBox:
				case ComponentType.FFLabel:
				case ComponentType.FFImage:
				case ComponentType.Image:
				case ComponentType.Slider:
				case ComponentType.SpinEdit:
				case ComponentType.TimeEdit:
				case ComponentType.WebComponent:
				case ComponentType.HLine:
				case ComponentType.DateTimeEdit:
					return Math.Max(1, num);
				case ComponentType.ScrollGrid:
					num = Math.Max(3, num);
					return Math.Max(this.GetContentWidth(), num);
				case ComponentType.Table:
				case ComponentType.Tree:
					return Math.Max(this.GetContentWidth(), num);
				}
				num = Math.Max(this.GetContentWidth(), num);
				return num;
			}
		}

		// Token: 0x0600036A RID: 874 RVA: 0x0000DE1C File Offset: 0x0000C01C
		private int GetContentWidth()
		{
			int num = 0;
			switch (this.Type)
			{
			case ComponentType.Folder:
			{
				using (IEnumerator<XmlElement> enumerator = this.Nodes.GetEnumerator())
				{
					while (enumerator.MoveNext())
					{
						XmlElement xmlElement = enumerator.Current;
						num = Math.Max(num, xmlElement.GetContentWidth());
					}
					goto IL_02B7;
				}
				goto IL_0227;
			}
			case ComponentType.Form:
			case ComponentType.Grid:
			case ComponentType.Page:
			case ComponentType.RadioGroup:
				goto IL_0272;
			case ComponentType.Group:
				goto IL_0227;
			case ComponentType.HBox:
			{
				for (int i = 0; i < this.Nodes.Count; i++)
				{
					XmlElement xmlElement2 = this.Nodes[i];
					num += ((xmlElement2.Index == this.Nodes.Count - 1) ? xmlElement2.MinGridWidth : xmlElement2.GridWidth);
				}
				num += 4;
				goto IL_02B7;
			}
			case ComponentType.ScrollGrid:
				foreach (XmlElement xmlElement3 in this.Nodes)
				{
					int num2 = xmlElement3.GridX + xmlElement3.GridWidth;
					if (string.Equals(xmlElement3.GetAttribute("repeat"), "true", StringComparison.CurrentCultureIgnoreCase))
					{
						int num3 = ((xmlElement3.ColumnCount == 0) ? 1 : xmlElement3.ColumnCount);
						num2 = num3 * xmlElement3.GridWidth + (xmlElement3.ColumnCount - 1) * xmlElement3.StepX + xmlElement3.GridX;
					}
					num = Math.Max(num, num2);
				}
				num++;
				goto IL_02B7;
			case ComponentType.Table:
			case ComponentType.Tree:
				break;
			case ComponentType.VBox:
			{
				using (IEnumerator<XmlElement> enumerator3 = this.Nodes.GetEnumerator())
				{
					while (enumerator3.MoveNext())
					{
						XmlElement xmlElement4 = enumerator3.Current;
						num = Math.Max(num, xmlElement4.MinGridWidth);
					}
					goto IL_02B7;
				}
				break;
			}
			default:
				goto IL_0272;
			}
			for (int j = 0; j < this.Nodes.Count; j++)
			{
				XmlElement xmlElement5 = this.Nodes[j];
				if (!(xmlElement5.NodeName == ComponentType.Phantom.ToString()))
				{
					num += xmlElement5.GridWidth;
					if (j > 0 && this.Nodes.Count > 1)
					{
						num++;
					}
				}
			}
			num += 2;
			goto IL_02B7;
			IL_0227:
			foreach (XmlElement xmlElement6 in this.Nodes)
			{
				num = Math.Max(num, xmlElement6.GridX + xmlElement6.GridWidth);
			}
			num++;
			goto IL_02B7;
			IL_0272:
			foreach (XmlElement xmlElement7 in this.Nodes)
			{
				num = Math.Max(num, xmlElement7.GridX + xmlElement7.GridWidth);
			}
			IL_02B7:
			ComponentType type = this.Type;
			switch (type)
			{
			case ComponentType.Folder:
			case ComponentType.Grid:
				break;
			case ComponentType.Form:
				return num;
			default:
				if (type != ComponentType.ScrollGrid)
				{
					return num;
				}
				break;
			}
			num++;
			return num;
		}

		// Token: 0x170000E6 RID: 230
		// (get) Token: 0x0600036B RID: 875 RVA: 0x0000E148 File Offset: 0x0000C348
		public int MinGridHeight
		{
			get
			{
				int num = 1;
				if (this.GetAttribute("minHeight") != null)
				{
					int.TryParse(this.GetAttribute("minHeight"), out num);
				}
				switch (this.Type)
				{
				case ComponentType.Folder:
					num = Math.Max(3, num);
					return Math.Max(this.GetContentHeight() + 1, num);
				case ComponentType.Form:
				case ComponentType.Grid:
				case ComponentType.ScrollGrid:
					num = Math.Max(2, num);
					return Math.Max(this.GetContentHeight(), num);
				case ComponentType.Group:
					num = Math.Max(2, num);
					return Math.Max(this.GetContentHeight() + 1, num);
				case ComponentType.HBox:
					num = Math.Max(3, num);
					return Math.Max(this.GetContentHeight(), num);
				case ComponentType.Page:
				case ComponentType.RadioGroupItem:
				case ComponentType.Phantom:
				case ComponentType.Item:
					return num;
				case ComponentType.RadioGroup:
				case ComponentType.Label:
				case ComponentType.Edit:
				case ComponentType.ProgressBar:
				case ComponentType.ComboBox:
				case ComponentType.TextEdit:
				case ComponentType.Button:
				case ComponentType.ButtonEdit:
				case ComponentType.DateEdit:
				case ComponentType.Canvas:
				case ComponentType.CheckBox:
				case ComponentType.FFLabel:
				case ComponentType.FFImage:
				case ComponentType.Image:
				case ComponentType.Slider:
				case ComponentType.SpinEdit:
				case ComponentType.TimeEdit:
				case ComponentType.WebComponent:
				case ComponentType.HLine:
				case ComponentType.DateTimeEdit:
					return 1;
				case ComponentType.Table:
				{
					num = this.TotalRows * this.RowHeight + 1;
					using (IEnumerator<XmlElement> enumerator = this.Nodes.GetEnumerator())
					{
						while (enumerator.MoveNext())
						{
							XmlElement xmlElement = enumerator.Current;
							if (xmlElement.IsEnabledAggregate)
							{
								num += this.RowHeight;
								break;
							}
						}
						return num;
					}
					break;
				}
				case ComponentType.Tree:
					break;
				case ComponentType.VBox:
					num = Math.Max(4, num);
					return Math.Max(this.GetContentHeight() + 1 + 1, num);
				default:
					return num;
				}
				num = this.TotalRows * this.RowHeight + 1;
				return num;
			}
		}

		// Token: 0x0600036C RID: 876 RVA: 0x0000E308 File Offset: 0x0000C508
		private int GetContentHeight()
		{
			int num = 0;
			switch (this.Type)
			{
			case ComponentType.Folder:
				break;
			case ComponentType.Form:
			case ComponentType.Grid:
			case ComponentType.Page:
			case ComponentType.RadioGroup:
				goto IL_022F;
			case ComponentType.Group:
				goto IL_01E4;
			case ComponentType.HBox:
			{
				using (IEnumerator<XmlElement> enumerator = this.Nodes.GetEnumerator())
				{
					while (enumerator.MoveNext())
					{
						XmlElement xmlElement = enumerator.Current;
						num = Math.Max(num, xmlElement.MinGridHeight);
					}
					goto IL_0274;
				}
				break;
			}
			case ComponentType.ScrollGrid:
			{
				using (IEnumerator<XmlElement> enumerator2 = this.Nodes.GetEnumerator())
				{
					while (enumerator2.MoveNext())
					{
						XmlElement xmlElement2 = enumerator2.Current;
						int num2 = xmlElement2.GridY + xmlElement2.GridHeight;
						if (string.Equals(xmlElement2.GetAttribute("repeat"), "true", StringComparison.CurrentCultureIgnoreCase))
						{
							num2 = xmlElement2.RowCount * xmlElement2.GridHeight + (xmlElement2.RowCount - 1) * xmlElement2.StepY + xmlElement2.GridY;
						}
						num = Math.Max(num, num2);
					}
					goto IL_0274;
				}
				goto IL_01E4;
			}
			case ComponentType.Table:
			case ComponentType.Tree:
				goto IL_0148;
			case ComponentType.VBox:
			{
				for (int i = 0; i < this.Nodes.Count; i++)
				{
					XmlElement xmlElement3 = this.Nodes[i];
					num += ((xmlElement3.Index == this.Nodes.Count - 1) ? xmlElement3.MinGridHeight : xmlElement3.GridHeight);
				}
				goto IL_0274;
			}
			default:
				goto IL_022F;
			}
			using (IEnumerator<XmlElement> enumerator3 = this.Nodes.GetEnumerator())
			{
				while (enumerator3.MoveNext())
				{
					XmlElement xmlElement4 = enumerator3.Current;
					foreach (XmlElement xmlElement5 in xmlElement4.Nodes)
					{
						num = Math.Max(num, xmlElement5.GridY + xmlElement5.GridHeight);
					}
				}
				goto IL_0274;
			}
			IL_0148:
			num = this.MinGridHeight;
			goto IL_0274;
			IL_01E4:
			foreach (XmlElement xmlElement6 in this.Nodes)
			{
				num = Math.Max(num, xmlElement6.GridY + xmlElement6.GridHeight);
			}
			num++;
			goto IL_0274;
			IL_022F:
			foreach (XmlElement xmlElement7 in this.Nodes)
			{
				num = Math.Max(num, xmlElement7.GridY + xmlElement7.GridHeight);
			}
			IL_0274:
			ComponentType type = this.Type;
			switch (type)
			{
			case ComponentType.Folder:
			case ComponentType.Grid:
				break;
			case ComponentType.Form:
				return num;
			default:
				if (type != ComponentType.ScrollGrid)
				{
					return num;
				}
				break;
			}
			num++;
			return num;
		}

		// Token: 0x170000E7 RID: 231
		// (get) Token: 0x0600036D RID: 877 RVA: 0x0000E5FC File Offset: 0x0000C7FC
		public int MinGridX
		{
			get
			{
				if (this.Parent == null)
				{
					return 0;
				}
				int num = 0;
				ComponentType type = this.Parent.Type;
				switch (type)
				{
				case ComponentType.Grid:
				case ComponentType.Group:
					break;
				default:
					if (type != ComponentType.ScrollGrid)
					{
						return num;
					}
					break;
				}
				num = 1;
				return num;
			}
		}

		// Token: 0x170000E8 RID: 232
		// (get) Token: 0x0600036E RID: 878 RVA: 0x0000E638 File Offset: 0x0000C838
		public int MinGridY
		{
			get
			{
				if (this.Parent == null)
				{
					return 0;
				}
				int num = 0;
				ComponentType type = this.Parent.Type;
				if (type == ComponentType.Group || type == ComponentType.ScrollGrid)
				{
					num = 1;
				}
				return num;
			}
		}

		// Token: 0x170000E9 RID: 233
		// (get) Token: 0x0600036F RID: 879 RVA: 0x0000E668 File Offset: 0x0000C868
		// (set) Token: 0x06000370 RID: 880 RVA: 0x0000E718 File Offset: 0x0000C918
		public string LocalString
		{
			get
			{
				string text = this.GetAttribute("text");
				if (this.Parent != null)
				{
					switch (this.Parent.Type)
					{
					case ComponentType.Table:
					case ComponentType.Tree:
						text = this.GetAttribute("title");
						break;
					}
				}
				if (text == null)
				{
					return null;
				}
				if (!string.IsNullOrEmpty(text))
				{
					ComponentType type = this.Type;
					if (type == ComponentType.RadioGroupItem)
					{
						text = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.GetItemLocalStringText(text);
					}
					else
					{
						text = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.GetFieldLocalStringText(text);
					}
				}
				if (string.IsNullOrEmpty(text))
				{
					return this.Name;
				}
				return text;
			}
			set
			{
				string text = this.GetAttribute("text");
				string attribute = this.GetAttribute("name");
				if (this.Parent != null && (this.Parent.Type == ComponentType.Table || this.Parent.Type == ComponentType.Tree))
				{
					text = this.GetAttribute("title");
					if (string.IsNullOrEmpty(text))
					{
						text = attribute;
						this.SetAttribute("title", attribute);
					}
				}
				if (text == null)
				{
					return;
				}
				SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.SetFieldLocalStringText(text, value);
				ComponentType type = this.Type;
				if (type == ComponentType.Button)
				{
					FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.FindNodeByName(this.Name);
					if (formSpecModel != null && formSpecModel.SpecAction != null)
					{
						formSpecModel.SpecAction.OnPropertyChanged("LocalString");
					}
				}
				this.OnPropertyChanged("LocalString");
			}
		}

		// Token: 0x170000EA RID: 234
		// (get) Token: 0x06000371 RID: 881 RVA: 0x0000E7F8 File Offset: 0x0000C9F8
		// (set) Token: 0x06000372 RID: 882 RVA: 0x0000E808 File Offset: 0x0000CA08
		public string Name
		{
			get
			{
				return this.GetAttribute("name");
			}
			set
			{
				if (value == this.Name)
				{
					return;
				}
				if ("s_browse" == this.Name || "s_browse" == value)
				{
					throw new Exception(Application.Current.FindResource("Message_TreeNameIllegal") as string);
				}
				RenameUndoRedoCommand renameUndoRedoCommand = new RenameUndoRedoCommand(this.Key, this.Name, value);
				SettingManager.Get().GetUndoRedoManager(this.Key).AddThenExecute(renameUndoRedoCommand);
			}
		}

		// Token: 0x170000EB RID: 235
		// (get) Token: 0x06000373 RID: 883 RVA: 0x0000E886 File Offset: 0x0000CA86
		// (set) Token: 0x06000374 RID: 884 RVA: 0x0000E88E File Offset: 0x0000CA8E
		public ComponentType Type { get; private set; }

		// Token: 0x06000375 RID: 885 RVA: 0x0000E898 File Offset: 0x0000CA98
		public XmlElement(PackageKey key, XElement source)
		{
			this.Key = key;
			this.Source = XElement.Parse(source.ToString());
			this.Source.RemoveNodes();
			this._nodeName = source.Name.LocalName;
			this.Type = XmlElement.GetComponentType(this.NodeName);
			if (this.NodeName == "DateTimeEdit" && SettingManager.Get().ErpVer == "1.0")
			{
				return;
			}
			ComponentFactory.AttachDefaultAttributes(key, this, false);
		}

		// Token: 0x06000376 RID: 886 RVA: 0x0000E92C File Offset: 0x0000CB2C
		public XmlElement(PackageKey key, ComponentType type)
		{
			this.Key = key;
			this._nodeName = type.ToString();
			this.Type = XmlElement.GetComponentType(this.NodeName);
			this.Source = new XElement(this._nodeName);
			ComponentFactory.AttachDefaultAttributes(key, this);
		}

		// Token: 0x06000377 RID: 887 RVA: 0x0000E990 File Offset: 0x0000CB90
		private static ComponentType GetComponentType(string nodeName)
		{
			switch (nodeName)
			{
			case "ButtonEdit":
				return ComponentType.ButtonEdit;
			case "CheckBox":
				return ComponentType.CheckBox;
			case "ComboBox":
				return ComponentType.ComboBox;
			case "DateEdit":
				return ComponentType.DateEdit;
			case "Edit":
				return ComponentType.Edit;
			case "FFImage":
				return ComponentType.FFImage;
			case "FFLabel":
				return ComponentType.FFLabel;
			case "ProgressBar":
				return ComponentType.ProgressBar;
			case "RadioGroup":
				return ComponentType.RadioGroup;
			case "Slider":
				return ComponentType.Slider;
			case "TextEdit":
				return ComponentType.TextEdit;
			case "TimeEdit":
				return ComponentType.TimeEdit;
			case "DateTimeEdit":
				return ComponentType.DateTimeEdit;
			case "WebComponent":
				return ComponentType.WebComponent;
			case "Button":
				return ComponentType.Button;
			case "ScrollGrid":
				return ComponentType.ScrollGrid;
			case "Canvas":
				return ComponentType.Canvas;
			case "Folder":
				return ComponentType.Folder;
			case "Form":
				return ComponentType.Form;
			case "Grid":
				return ComponentType.Grid;
			case "Group":
				return ComponentType.Group;
			case "HBox":
				return ComponentType.HBox;
			case "HLine":
				return ComponentType.HLine;
			case "Image":
				return ComponentType.Image;
			case "SpinEdit":
				return ComponentType.SpinEdit;
			case "Table":
				return ComponentType.Table;
			case "Tree":
				return ComponentType.Tree;
			case "VBox":
				return ComponentType.VBox;
			case "Label":
				return ComponentType.Label;
			case "Phantom":
				return ComponentType.Phantom;
			case "Page":
				return ComponentType.Page;
			case "Item":
				return ComponentType.Item;
			}
			return ComponentType.Unknown;
		}

		// Token: 0x170000EC RID: 236
		// (get) Token: 0x06000378 RID: 888 RVA: 0x0000EC4B File Offset: 0x0000CE4B
		public ObservableCollection<XmlElement> Nodes
		{
			get
			{
				if (this._nodes == null)
				{
					this._nodes = new ObservableCollection<XmlElement>();
				}
				return this._nodes;
			}
		}

		// Token: 0x170000ED RID: 237
		// (get) Token: 0x06000379 RID: 889 RVA: 0x0000EC66 File Offset: 0x0000CE66
		public bool HasNodes
		{
			get
			{
				return this.Nodes.Count<XmlElement>() > 0;
			}
		}

		// Token: 0x170000EE RID: 238
		// (get) Token: 0x0600037A RID: 890 RVA: 0x0000EC76 File Offset: 0x0000CE76
		public int Index
		{
			get
			{
				if (this.Parent != null)
				{
					return this.Parent.Nodes.IndexOf(this);
				}
				return 0;
			}
		}

		// Token: 0x0600037B RID: 891 RVA: 0x0000EC93 File Offset: 0x0000CE93
		public void AddNode(XmlElement element)
		{
			this.AddNodeAt(element, 0);
		}

		// Token: 0x0600037C RID: 892 RVA: 0x0000ECA0 File Offset: 0x0000CEA0
		public void AddNodeAt(XmlElement element, int index)
		{
			if (element.Parent == this && index == element.Index)
			{
				return;
			}
			index = ((index < this.Nodes.Count) ? index : this.Nodes.Count);
			bool flag = element.Parent != this;
			if (flag && element.Parent != null)
			{
				element.Parent.RemoveNode(element);
			}
			if (flag)
			{
				element.Parent = this;
				this.Nodes.Insert(index, element);
				ComponentType type = this.Type;
				if (type == ComponentType.ScrollGrid)
				{
					if (element.GetAttribute("repeat") != null)
					{
						element.SetAttribute("repeat", "true");
						if ("" == element.GetAttribute("stepX") || XmlElement.NOSET == element.GetAttribute("stepX"))
						{
							element.SetAttribute("stepX", "1");
						}
						if ("" == element.GetAttribute("stepY") || XmlElement.NOSET == element.GetAttribute("stepY"))
						{
							element.SetAttribute("stepY", "0");
						}
						if ("" == element.GetAttribute("columnCount") || XmlElement.NOSET == element.GetAttribute("columnCount"))
						{
							element.SetAttribute("columnCount", "2");
						}
						if ("" == element.GetAttribute("rowCount") || XmlElement.NOSET == element.GetAttribute("rowCount"))
						{
							element.SetAttribute("rowCount", "1");
						}
					}
				}
				else if (element.GetAttribute("repeat") != null)
				{
					element.RemoveAttribute("repeat");
				}
				ComponentType type2 = this.Type;
				if (type2 != ComponentType.HBox)
				{
					if (type2 != ComponentType.VBox)
					{
						element.GridX = Math.Max(element.GridX, element.MinGridX);
						element.GridY = Math.Max(element.GridY, element.MinGridY);
					}
					else
					{
						element.GridWidth = Math.Max(element.GridWidth, this.GridWidth);
						element.GridX = 0;
					}
				}
				else
				{
					element.GridHeight = Math.Max(element.GridHeight, this.GridHeight);
					element.GridY = 0;
				}
				if (this.Name == "s_browse")
				{
					FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.FindNodeByName(this.Name);
					if (formSpecModel.SpecField != null)
					{
						formSpecModel.SpecField.SetAttribute("can_edit", "N");
						formSpecModel.SpecField.SetAttribute("can_query", "N");
					}
				}
				this.MeasureSize();
			}
			else
			{
				this.Nodes.Move(element.Index, index);
			}
			this.MeasurePos();
			this.CheckOverlapping();
		}

		// Token: 0x0600037D RID: 893 RVA: 0x0000EF68 File Offset: 0x0000D168
		internal void MeasurePos()
		{
			switch (this.Type)
			{
			case ComponentType.HBox:
			{
				int num = 2;
				for (int i = 0; i < this.Nodes.Count; i++)
				{
					XmlElement xmlElement = this.Nodes[i];
					xmlElement.GridX = num;
					num += xmlElement.GridWidth;
				}
				return;
			}
			case ComponentType.Page:
			case ComponentType.RadioGroup:
			case ComponentType.ScrollGrid:
				break;
			case ComponentType.Table:
			case ComponentType.Tree:
			{
				int num2 = 0;
				for (int j = 0; j < this.Nodes.Count; j++)
				{
					XmlElement xmlElement2 = this.Nodes[j];
					xmlElement2.GridX = num2;
					num2 += xmlElement2.GridWidth;
				}
				return;
			}
			case ComponentType.VBox:
			{
				int num3 = 1;
				for (int k = 0; k < this.Nodes.Count; k++)
				{
					XmlElement xmlElement3 = this.Nodes[k];
					xmlElement3.GridY = num3;
					num3 += xmlElement3.GridHeight;
				}
				break;
			}
			default:
				return;
			}
		}

		// Token: 0x0600037E RID: 894 RVA: 0x0000F058 File Offset: 0x0000D258
		internal void MeasureSize()
		{
			bool flag = true;
			bool flag2 = true;
			int num = 0;
			int num2 = 0;
			bool flag3 = false;
			ComponentType type = this.Type;
			if (type != ComponentType.Folder)
			{
				switch (type)
				{
				case ComponentType.HBox:
					if (flag)
					{
						num = 0;
						num2 = this.GridWidth;
						foreach (XmlElement xmlElement in this.Nodes)
						{
							num += xmlElement.GridWidth;
						}
						this.GridWidth = Math.Max(this.MinGridWidth, num + 4);
						flag3 = this.GridWidth > num2;
					}
					else
					{
						num = 0;
						num2 = this.GridWidth;
						foreach (XmlElement xmlElement2 in this.Nodes)
						{
							num += xmlElement2.GridWidth;
						}
						this.GridWidth = Math.Max(this.GridWidth, num + 4);
						flag3 = this.GridWidth > num2;
					}
					num = 0;
					num2 = this.GridHeight;
					foreach (XmlElement xmlElement3 in this.Nodes)
					{
						num = Math.Max(xmlElement3.GridHeight, num);
					}
					this.GridHeight = Math.Max(this.GridHeight, num);
					this.MeasurePos();
					flag3 = true;
					goto IL_037F;
				case ComponentType.Page:
					flag3 = true;
					goto IL_037F;
				case ComponentType.Table:
					num2 = this.GridHeight;
					this.GridHeight = Math.Max(this.GridHeight, this.GetContentHeight());
					flag3 = this.GridHeight > num2;
					num2 = this.GridWidth;
					this.GridWidth = Math.Max(this.GridWidth, this.GetContentWidth());
					flag3 |= this.GridWidth > num2;
					goto IL_037F;
				case ComponentType.VBox:
					num = 0;
					num2 = this.GridWidth;
					foreach (XmlElement xmlElement4 in this.Nodes)
					{
						num = Math.Max(xmlElement4.GridWidth, num);
					}
					this.GridWidth = Math.Max(this.MinGridWidth, num);
					flag3 = this.GridWidth > num2;
					if (flag2)
					{
						num = 0;
						num2 = this.GridHeight;
						foreach (XmlElement xmlElement5 in this.Nodes)
						{
							num += xmlElement5.GridHeight;
						}
						this.GridHeight = Math.Max(this.MinGridHeight, num + 2);
						flag3 |= this.GridHeight > num2;
					}
					this.MeasurePos();
					goto IL_037F;
				}
				num2 = this.GridWidth;
				this.GridWidth = Math.Max(this.GridWidth, this.GetContentWidth());
				flag3 = this.GridWidth > num2;
				num2 = this.GridHeight;
				this.GridHeight = Math.Max(this.GridHeight, this.GetContentHeight());
				flag3 |= this.GridHeight > num2;
			}
			else
			{
				num2 = this.GridWidth;
				this.GridWidth = Math.Max(this.GridWidth, this.GetContentWidth());
				flag3 = this.GridWidth > num2;
				num2 = this.GridHeight;
				this.GridHeight = Math.Max(this.GridHeight, this.GetContentHeight() + 1);
				flag3 |= this.GridHeight > num2;
			}
			IL_037F:
			if (this.Parent != null && flag3)
			{
				Trace.WriteLine(string.Format("[{0}]: GridWidth: {1}, GridHeight: {2}", this.Name, this.GridWidth, this.GridHeight));
				this.Parent.MeasureSize();
			}
		}

		// Token: 0x0600037F RID: 895 RVA: 0x0000F468 File Offset: 0x0000D668
		public void MoveTo(XmlElement element, int index)
		{
			this.RemoveNode(element);
			this.AddNodeAt(element, index);
		}

		// Token: 0x06000380 RID: 896 RVA: 0x0000F479 File Offset: 0x0000D679
		public void RemoveNode(XmlElement element)
		{
			this.RemoveNode(element, true);
		}

		// Token: 0x06000381 RID: 897 RVA: 0x0000F483 File Offset: 0x0000D683
		public void RemoveNode(XmlElement element, bool needResize)
		{
			if (this._nodes != null)
			{
				element.Parent = null;
				this.Nodes.Remove(element);
				if (needResize)
				{
					this.MeasurePos();
					this.MeasureSize();
				}
				this.CheckOverlapping();
			}
		}

		// Token: 0x06000382 RID: 898 RVA: 0x0000F530 File Offset: 0x0000D730
		public void SortNodes()
		{
			if (FormDesignSetting.CanSortNodes(this.NodeName))
			{
				List<XmlElement> list = this.Nodes.ToList<XmlElement>();
				list.Sort(delegate(XmlElement a, XmlElement b)
				{
					try
					{
						int gridX = a.GridX;
						int gridY = a.GridY;
						int gridX2 = b.GridX;
						int gridY2 = b.GridY;
						if (gridY < gridY2)
						{
							return -1;
						}
						if (gridY == gridY2)
						{
							if (gridX < gridX2)
							{
								return -1;
							}
							if (gridX == gridX2)
							{
								return 0;
							}
							if (gridX > gridX2)
							{
								return 1;
							}
						}
						if (gridY > gridY2)
						{
							return 1;
						}
					}
					catch
					{
						return 0;
					}
					return 0;
				});
				for (int i = 0; i < list.Count; i++)
				{
					this.AddNodeAt(list[i], i);
				}
			}
		}

		// Token: 0x170000EF RID: 239
		// (get) Token: 0x06000383 RID: 899 RVA: 0x0000F598 File Offset: 0x0000D798
		public bool HasItems
		{
			get
			{
				return this._items != null && this._items.Count > 0;
			}
		}

		// Token: 0x170000F0 RID: 240
		// (get) Token: 0x06000384 RID: 900 RVA: 0x0000F5B2 File Offset: 0x0000D7B2
		public ObservableCollection<XmlElement> Items
		{
			get
			{
				if (this._items == null)
				{
					this._items = new ObservableCollection<XmlElement>();
				}
				return this._items;
			}
		}

		// Token: 0x06000385 RID: 901 RVA: 0x0000F5CD File Offset: 0x0000D7CD
		public string GetAttribute(string key)
		{
			if (this.Source.Attribute(key) == null)
			{
				return null;
			}
			return this.Source.Attribute(key).Value;
		}

		// Token: 0x06000386 RID: 902 RVA: 0x0000F5FC File Offset: 0x0000D7FC
		public void SetAttribute(string key, string value)
		{
			if (this.Source.Attribute(key) == null)
			{
				this.Source.SetAttributeValue(key, "");
			}
			this.Source.SetAttributeValue(key, value);
			string text;
			switch (text = key.ToLower())
			{
			case "gridwidth":
			case "gridheight":
			case "width":
			case "height":
			case "posx":
			case "posy":
				if (this.Parent != null)
				{
					this.Parent.CheckOverlapping();
				}
				break;

				return;
			}
		}

		// Token: 0x06000387 RID: 903 RVA: 0x0000F6F2 File Offset: 0x0000D8F2
		public void RemoveAttribute(string key)
		{
			if (this.Source.Attribute(key) != null)
			{
				this.Source.Attribute(key).Remove();
			}
		}

		// Token: 0x170000F1 RID: 241
		public string this[string key]
		{
			get
			{
				string attribute = this.GetAttribute(key);
				if (attribute != null)
				{
					if (!(attribute == XmlElement.NOSET))
					{
						return attribute;
					}
					return "";
				}
				else
				{
					if (key == "format" && (this.Type == ComponentType.Edit || this.Type == ComponentType.ButtonEdit || this.Type == ComponentType.DateEdit || this.Type == ComponentType.FFLabel || this.Type == ComponentType.SpinEdit || this.Type == ComponentType.TimeEdit || this.Type == ComponentType.DateTimeEdit))
					{
						this.SetAttribute("format", "");
						return "";
					}
					return null;
				}
			}
			set
			{
				string attribute = this.GetAttribute(key);
				if (attribute == null || attribute == value)
				{
					return;
				}
				switch (key)
				{
				case "tabIndex":
					this.SetAttribute(key, value);
					this.OnPropertyChanged("");
					return;
				case "repeat":
					if (this.Parent != null && this.Parent.Type == ComponentType.ScrollGrid && value != "true")
					{
						throw new InvalidOperationException(Application.Current.FindResource("Message_RepeatInScrollGrid") as string);
					}
					break;
				case "stepX":
				case "stepY":
					if (this.Parent == null || this.Parent.Type != ComponentType.ScrollGrid)
					{
						return;
					}
					break;
				case "rowCount":
				{
					if (this.Parent == null || this.Parent.Type != ComponentType.ScrollGrid)
					{
						return;
					}
					int num2 = 1;
					if (int.TryParse(value, out num2))
					{
						num2 = Math.Max(num2, 3 - this.ColumnCount);
						value = num2.ToString();
					}
					break;
				}
				case "columnCount":
				{
					if (this.Parent == null || this.Parent.Type != ComponentType.ScrollGrid)
					{
						return;
					}
					int num3 = 1;
					if (int.TryParse(value, out num3))
					{
						num3 = Math.Max(num3, 3 - this.ColumnCount);
						value = num3.ToString();
					}
					break;
				}
				case "gridWidth":
				{
					int minGridWidth = this.MinGridWidth;
					if (int.TryParse(value, out minGridWidth))
					{
						value = ((this.MinGridWidth < minGridWidth) ? minGridWidth : this.MinGridWidth).ToString();
					}
					break;
				}
				case "gridHeight":
				{
					int minGridHeight = this.MinGridHeight;
					if (int.TryParse(value, out minGridHeight))
					{
						value = ((this.MinGridHeight < minGridHeight) ? minGridHeight : this.MinGridHeight).ToString();
					}
					break;
				}
				case "posX":
				{
					int minGridX = this.MinGridX;
					if (int.TryParse(value, out minGridX))
					{
						value = ((this.MinGridX < minGridX) ? minGridX : this.MinGridX).ToString();
					}
					break;
				}
				case "posY":
				{
					int minGridY = this.MinGridY;
					if (int.TryParse(value, out minGridY))
					{
						value = ((this.MinGridY < minGridY) ? minGridY : this.MinGridY).ToString();
					}
					break;
				}
				}
				if (this.IsBatchSet)
				{
					if (key == "hidden")
					{
						this.SetAttribute(key, value);
					}
					return;
				}
				FormAttributesUndoRedoCommand formAttributesUndoRedoCommand = new FormAttributesUndoRedoCommand(this, key, value);
				formAttributesUndoRedoCommand.Execute();
			}
		}

		// Token: 0x0600038A RID: 906 RVA: 0x0000FAB8 File Offset: 0x0000DCB8
		public bool HasCantDelTags(string tag)
		{
			if (string.IsNullOrEmpty(tag))
			{
				return false;
			}
			List<string> list = tag.ToLower().Replace(" ", "").Split(new char[] { ',' })
				.ToList<string>();
			return list.Contains("cantdel") || list.Contains("commonpage") || list.Contains("formroot") || list.Contains("cantdel_formroot");
		}

		// Token: 0x170000F2 RID: 242
		// (get) Token: 0x0600038B RID: 907 RVA: 0x0000FCD4 File Offset: 0x0000DED4
		internal IEnumerable<string> Attributes
		{
			get
			{
				foreach (XAttribute att in this.Source.Attributes())
				{
					yield return att.Name.LocalName;
				}
				yield break;
			}
		}

		// Token: 0x0600038C RID: 908 RVA: 0x0000FCF1 File Offset: 0x0000DEF1
		public override string ToString()
		{
			return this.ToXML().ToString();
		}

		// Token: 0x0600038D RID: 909 RVA: 0x0000FD00 File Offset: 0x0000DF00
		public XElement ToXML()
		{
			this.SortNodes();
			XElement xelement = new XElement(this.NodeName);
			if ((ComponentType.ComboBox.ToString().Equals(this.NodeName) || ComponentType.RadioGroup.ToString().Equals(this.NodeName)) && this.GetAttribute("fieldType") != null && !"NON_DATABASE".Equals(this.GetAttribute("fieldType")))
			{
				this.Nodes.Clear();
				if (this.GetAttribute("items") != null)
				{
					this.SetAttribute("items", string.Empty);
				}
			}
			foreach (XAttribute xattribute in this.Source.Attributes())
			{
				string localName;
				switch (localName = xattribute.Name.LocalName)
				{
				case "unhidable":
				case "unmovable":
				case "unsizable":
				case "unsortable":
				case "picture":
				case "style":
				case "format":
				case "width":
				case "height":
					if (string.IsNullOrEmpty(xattribute.Value))
					{
						continue;
					}
					break;
				case "repeat":
					if (this.Parent != null && this.Parent.NodeName != ComponentType.ScrollGrid.ToString())
					{
						continue;
					}
					break;
				case "fontPitch":
				case "sqlType":
				case "lookup":
				case "sample":
				case "ref_usage":
				case "guid":
				case "displayTabName":
				case "displayColName":
				case "validateTabName":
				case "validateColName":
					continue;
				case "gridHeight":
					if (string.IsNullOrEmpty(xattribute.Value))
					{
						xattribute.Value = "1";
					}
					break;
				}
				if (!(xattribute.Value == XmlElement.NOSET))
				{
					xelement.Add(xattribute);
				}
			}
			if (this.Nodes != null)
			{
				foreach (XmlElement xmlElement in this.Nodes)
				{
					xelement.Add(xmlElement.ToXML());
				}
			}
			ComponentType type = this.Type;
			if ((type == ComponentType.RadioGroup || type == ComponentType.ComboBox) && this._items != null)
			{
				foreach (XmlElement xmlElement2 in this._items)
				{
					xelement.Add(xmlElement2.ToXML());
				}
			}
			if (this.Type == ComponentType.ButtonEdit)
			{
				XmlElement.AddAttributeCompleter(xelement);
			}
			return xelement;
		}

		// Token: 0x0600038E RID: 910 RVA: 0x000100E0 File Offset: 0x0000E2E0
		public static void BatchAddAttributeCompleter(XElement form)
		{
			if (form != null)
			{
				IEnumerable<XElement> enumerable = from item in form.Descendants("ButtonEdit")
					select (item);
				foreach (XElement xelement in enumerable)
				{
					XmlElement.AddAttributeCompleter(xelement);
				}
			}
		}

		// Token: 0x0600038F RID: 911 RVA: 0x00010160 File Offset: 0x0000E360
		public static void AddAttributeCompleter(XElement xe)
		{
			string text = (string)xe.Attribute("sqlTabName");
			string text2 = (string)xe.Attribute("colName");
			string text3 = (string)xe.Attribute("completer");
			if (!string.IsNullOrEmpty(text) && !string.IsNullOrEmpty(text2))
			{
				if (text3 == null)
				{
					xe.Add(new XAttribute("completer", "false"));
					return;
				}
				if (text3 != "true")
				{
					xe.SetAttributeValue("completer", "false");
					return;
				}
			}
			else if (string.IsNullOrEmpty(text) && string.IsNullOrEmpty(text2) && text3 == "true")
			{
				xe.SetAttributeValue("completer", "false");
			}
		}

		// Token: 0x1400000D RID: 13
		// (add) Token: 0x06000390 RID: 912 RVA: 0x00010234 File Offset: 0x0000E434
		// (remove) Token: 0x06000391 RID: 913 RVA: 0x0001026C File Offset: 0x0000E46C
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000392 RID: 914 RVA: 0x000102A4 File Offset: 0x0000E4A4
		public void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
			if ((propertyName == null || (!(propertyName == "IsSelected") && !(propertyName == "IsShowTabIndex") && !(propertyName == "IsFocused") && !(propertyName == "IsDragOver") && !(propertyName == "IsExpanded"))) && EventAggregatorManager.ContainsKey(this.Key))
			{
				EventAggregatorManager.Get(this.Key).GetEvent<FormPropertyChangedEvent>().Publish(this.Key);
			}
		}

		// Token: 0x06000393 RID: 915 RVA: 0x0001033C File Offset: 0x0000E53C
		public List<XmlElement> GetItems()
		{
			ComponentType type = this.Type;
			if (type == ComponentType.RadioGroup || type == ComponentType.ComboBox)
			{
				return this.Items.ToList<XmlElement>();
			}
			return null;
		}

		// Token: 0x170000F3 RID: 243
		// (get) Token: 0x06000394 RID: 916 RVA: 0x00010500 File Offset: 0x0000E700
		public IEnumerable<string> ChildrenNames
		{
			get
			{
				foreach (XmlElement child in this.Nodes)
				{
					yield return child.Name;
				}
				yield break;
			}
		}

		// Token: 0x06000395 RID: 917 RVA: 0x00010520 File Offset: 0x0000E720
		public void ReplaceItems(List<XmlElement> newItems)
		{
			ComponentType type = this.Type;
			if (type != ComponentType.RadioGroup && type != ComponentType.ComboBox)
			{
				return;
			}
			this.Items.Clear();
			List<string> list = new List<string>();
			foreach (XmlElement xmlElement in newItems)
			{
				this.Items.Add(xmlElement);
				list.Add(xmlElement.Text);
			}
			this.SetAttribute("items", string.Join(", ", list));
			this.OnPropertyChanged("");
			this.OnPropertyChanged("HasItems");
		}

		// Token: 0x170000F4 RID: 244
		// (get) Token: 0x06000396 RID: 918 RVA: 0x000105D0 File Offset: 0x0000E7D0
		public bool CanDelIncludeChildren
		{
			get
			{
				if (this.NodeName == ComponentType.Form.ToString())
				{
					return false;
				}
				if (this.Parent != null && this.Parent.NodeName == ComponentType.Form.ToString())
				{
					return false;
				}
				if (this.IsCantDel)
				{
					return false;
				}
				foreach (XmlElement xmlElement in this.Nodes)
				{
					if (!xmlElement.CanDelIncludeChildren)
					{
						return false;
					}
				}
				return true;
			}
		}

		// Token: 0x170000F5 RID: 245
		// (get) Token: 0x06000397 RID: 919 RVA: 0x00010670 File Offset: 0x0000E870
		public bool IsFormField
		{
			get
			{
				return this.GetAttribute("fieldId") != null;
			}
		}

		// Token: 0x06000398 RID: 920 RVA: 0x00010684 File Offset: 0x0000E884
		public ResizeDirectionEnum GetGeneroResizeDirection()
		{
			ResizeDirectionEnum resizeDirectionEnum = ResizeDirectionEnum.NONE;
			if (this.Parent != null)
			{
				switch (this.Parent.Type)
				{
				case ComponentType.HBox:
					if (this.Index == this.Parent.Nodes.Count - 1)
					{
						return ResizeDirectionEnum.NONE;
					}
					return ResizeDirectionEnum.RIGHT;
				case ComponentType.Table:
				case ComponentType.Tree:
					return ResizeDirectionEnum.RIGHT;
				case ComponentType.VBox:
					if (this.Index == this.Parent.Nodes.Count - 1)
					{
						return ResizeDirectionEnum.NONE;
					}
					return ResizeDirectionEnum.BOTTOM;
				}
			}
			switch (this.Type)
			{
			case ComponentType.Folder:
			case ComponentType.Form:
			case ComponentType.Grid:
			case ComponentType.Group:
			case ComponentType.HBox:
			case ComponentType.ScrollGrid:
			case ComponentType.Table:
			case ComponentType.Tree:
			case ComponentType.VBox:
				resizeDirectionEnum = ResizeDirectionEnum.BOTH;
				break;
			case ComponentType.Page:
				resizeDirectionEnum = ResizeDirectionEnum.NONE;
				break;
			case ComponentType.RadioGroup:
			case ComponentType.Label:
			case ComponentType.Edit:
			case ComponentType.ProgressBar:
			case ComponentType.ComboBox:
			case ComponentType.TextEdit:
			case ComponentType.Button:
			case ComponentType.ButtonEdit:
			case ComponentType.DateEdit:
			case ComponentType.Canvas:
			case ComponentType.CheckBox:
			case ComponentType.FFLabel:
			case ComponentType.FFImage:
			case ComponentType.Image:
			case ComponentType.Slider:
			case ComponentType.SpinEdit:
			case ComponentType.TimeEdit:
			case ComponentType.WebComponent:
			case ComponentType.HLine:
			case ComponentType.DateTimeEdit:
				resizeDirectionEnum = ResizeDirectionEnum.BOTH;
				break;
			}
			return resizeDirectionEnum;
		}

		// Token: 0x170000F6 RID: 246
		// (get) Token: 0x06000399 RID: 921 RVA: 0x000107AB File Offset: 0x0000E9AB
		// (set) Token: 0x0600039A RID: 922 RVA: 0x000107B3 File Offset: 0x0000E9B3
		public bool IsExpanded
		{
			get
			{
				return this._isExpanded;
			}
			set
			{
				this._isExpanded = value;
				this.OnPropertyChanged("IsExpanded");
			}
		}

		// Token: 0x0600039B RID: 923 RVA: 0x000107C8 File Offset: 0x0000E9C8
		public static void ExpandAll(XmlElement xe, bool val)
		{
			if (xe != null)
			{
				xe.IsExpanded = val;
				foreach (XmlElement xmlElement in xe.Nodes)
				{
					XmlElement.ExpandAll(xmlElement, val);
				}
			}
		}

		// Token: 0x0600039C RID: 924 RVA: 0x00010820 File Offset: 0x0000EA20
		public static void ExpandCurrentChild(XmlElement xe, bool val)
		{
			if (xe != null)
			{
				foreach (XmlElement xmlElement in xe.Nodes)
				{
					xmlElement.IsExpanded = val;
				}
			}
		}

		// Token: 0x0600039D RID: 925 RVA: 0x00010870 File Offset: 0x0000EA70
		public bool Contains(string tag, string value)
		{
			foreach (string text in this.Attributes)
			{
				if ((tag == null || !(text != tag)) && this.GetAttribute(text) == value)
				{
					return true;
				}
			}
			return false;
		}

		// Token: 0x0600039E RID: 926 RVA: 0x000108D8 File Offset: 0x0000EAD8
		public void ChangeAttributeValue(string tag, string oldValue, string newValue)
		{
			foreach (string text in this.Attributes)
			{
				if ((tag == null || !(text != tag)) && this.GetAttribute(text) == oldValue)
				{
					this.SetAttribute(tag, newValue);
				}
			}
		}

		// Token: 0x0600039F RID: 927 RVA: 0x00010944 File Offset: 0x0000EB44
		public static XmlElement Create(PackageKey key, string source)
		{
			XElement xelement = null;
			try
			{
				xelement = XElement.Parse(source);
			}
			catch
			{
				xelement = null;
			}
			XmlElement xmlElement = ((xelement == null) ? null : new XmlElement(key, xelement));
			ComponentType type = xmlElement.Type;
			if (type == ComponentType.RadioGroup || type == ComponentType.ComboBox)
			{
				foreach (XElement xelement2 in xelement.Elements())
				{
					xmlElement.Items.Add(new XmlElement(key, xelement2));
				}
			}
			return xmlElement;
		}

		// Token: 0x170000F7 RID: 247
		// (get) Token: 0x060003A0 RID: 928 RVA: 0x000109DC File Offset: 0x0000EBDC
		// (set) Token: 0x060003A1 RID: 929 RVA: 0x000109E4 File Offset: 0x0000EBE4
		public bool IsDragOver
		{
			get
			{
				return this._isDragOver;
			}
			set
			{
				if (this._isDragOver == value)
				{
					return;
				}
				this._isDragOver = value;
				this.OnPropertyChanged("IsDragOver");
			}
		}

		// Token: 0x170000F8 RID: 248
		// (get) Token: 0x060003A2 RID: 930 RVA: 0x00010A0C File Offset: 0x0000EC0C
		public ICommand AppendSelectedCommand
		{
			get
			{
				if (this._appendSelectedCommand == null)
				{
					this._appendSelectedCommand = new RelayCommand(delegate(object p)
					{
						this.OnAppendSelected();
					});
				}
				return this._appendSelectedCommand;
			}
		}

		// Token: 0x060003A3 RID: 931 RVA: 0x00010A48 File Offset: 0x0000EC48
		private void OnAppendSelected()
		{
			if (ComponentHelper.Get(this.Key).SelectedObjects.Contains(this))
			{
				if ((Keyboard.Modifiers & ModifierKeys.Control) == ModifierKeys.Control)
				{
					ComponentHelper.Get(this.Key).AddSelection(this, true);
				}
				return;
			}
			ComponentHelper.Get(this.Key).AddSelection(this, (Keyboard.Modifiers & ModifierKeys.Control) == ModifierKeys.Control);
		}

		// Token: 0x170000F9 RID: 249
		// (get) Token: 0x060003A4 RID: 932 RVA: 0x00010AB0 File Offset: 0x0000ECB0
		public ICommand SelectedSelfCommand
		{
			get
			{
				if (this._selectedSelfCommand == null)
				{
					this._selectedSelfCommand = new RelayCommand(delegate(object p)
					{
						this.OnSelectedSelf();
					});
				}
				return this._selectedSelfCommand;
			}
		}

		// Token: 0x060003A5 RID: 933 RVA: 0x00010AE9 File Offset: 0x0000ECE9
		private void OnSelectedSelf()
		{
			if (!ComponentHelper.Get(this.Key).SelectionContains(this))
			{
				ComponentHelper.Get(this.Key).AddSelection(this, false);
			}
		}

		// Token: 0x060003A6 RID: 934 RVA: 0x00010B10 File Offset: 0x0000ED10
		public void UpdateAggregateStatus()
		{
			if (this.Type != ComponentType.Table)
			{
				return;
			}
			foreach (XmlElement xmlElement in this.Nodes)
			{
				xmlElement.OnPropertyChanged("IsAnySiblingEnabledAggregate");
			}
			this.MeasureSize();
		}

		// Token: 0x170000FA RID: 250
		// (get) Token: 0x060003A7 RID: 935 RVA: 0x00010B74 File Offset: 0x0000ED74
		// (set) Token: 0x060003A8 RID: 936 RVA: 0x00010B7C File Offset: 0x0000ED7C
		public bool IsPosFine
		{
			get
			{
				return this._isPosFine;
			}
			set
			{
				if (value != this._isPosFine)
				{
					this._isPosFine = value;
					this.OnPropertyChanged("IsPosFine");
				}
			}
		}

		// Token: 0x060003A9 RID: 937 RVA: 0x00010B9C File Offset: 0x0000ED9C
		public void CheckOverlapping()
		{
			if (this.GridHeight == 0 || this.GridWidth == 0)
			{
				return;
			}
			if (!PreferenceManager.Current.Settings.ValidateForm)
			{
				return;
			}
			switch (this.Type)
			{
			case ComponentType.HBox:
			case ComponentType.Page:
			case ComponentType.Table:
			case ComponentType.Tree:
			case ComponentType.VBox:
				return;
			}
			ContainerMapManager.This.CheckOverlapping(this);
		}

		// Token: 0x170000FB RID: 251
		// (get) Token: 0x060003AA RID: 938 RVA: 0x00010C05 File Offset: 0x0000EE05
		// (set) Token: 0x060003AB RID: 939 RVA: 0x00010C10 File Offset: 0x0000EE10
		public XmlElement BindElement
		{
			get
			{
				return this._BindElement;
			}
			set
			{
				this._BindElement = value;
				this.OnPropertyChanged("BindElement");
				this.OnPropertyChanged("HasElementBinding");
				if (value == null)
				{
					SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo;
					if (specificationInfo != null)
					{
						FormSpecModel formSpecModel = specificationInfo.FindNodeByName(this.Name);
						if (formSpecModel != null && formSpecModel.SpecField != null)
						{
							formSpecModel.SpecField.OnPropertyChanged("IsReadOnly");
						}
					}
				}
			}
		}

		// Token: 0x170000FC RID: 252
		// (get) Token: 0x060003AC RID: 940 RVA: 0x00010C7E File Offset: 0x0000EE7E
		public bool HasElementBinding
		{
			get
			{
				return this.BindElement != null;
			}
		}

		// Token: 0x060003AD RID: 941 RVA: 0x00010C8C File Offset: 0x0000EE8C
		public void ClearBinding()
		{
			XmlElement xmlElement = null;
			if (this.BindElement != null)
			{
				xmlElement = this.BindElement;
				FJSObject fjsobject = new FJSObject
				{
					Text = this.Name,
					Type = this.Type
				};
				FJSObject fjsobject2 = new FJSObject
				{
					Text = this.BindElement.Name,
					Type = this.BindElement.Type
				};
				TBinding tbinding = new TBinding
				{
					ObjectName1 = fjsobject,
					ObjectName2 = fjsobject2
				};
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo;
				if (specificationInfo.SpecBinding.Contains(tbinding))
				{
					specificationInfo.SpecBinding.Remove(tbinding);
				}
				int num = 1;
				foreach (TBinding tbinding2 in specificationInfo.SpecBinding)
				{
					tbinding2.ID = num++;
				}
			}
			this.BindElement = null;
			if (xmlElement != null)
			{
				xmlElement.ClearBinding();
			}
		}

		// Token: 0x0400012E RID: 302
		public static readonly string NOSET = "[[RE][SET]]";

		// Token: 0x0400012F RID: 303
		private XmlElement _parent;

		// Token: 0x04000130 RID: 304
		private string _nodeName;

		// Token: 0x04000131 RID: 305
		private bool isHidden;

		// Token: 0x04000132 RID: 306
		private bool _isShowTabIndex;

		// Token: 0x04000133 RID: 307
		private bool _isSelected;

		// Token: 0x04000134 RID: 308
		private bool _isFocused;

		// Token: 0x04000135 RID: 309
		private ObservableCollection<XmlElement> _nodes;

		// Token: 0x04000136 RID: 310
		private ObservableCollection<XmlElement> _items;

		// Token: 0x04000138 RID: 312
		private bool _isExpanded;

		// Token: 0x04000139 RID: 313
		private bool _isDragOver;

		// Token: 0x0400013A RID: 314
		private RelayCommand _appendSelectedCommand;

		// Token: 0x0400013B RID: 315
		private RelayCommand _selectedSelfCommand;

		// Token: 0x0400013C RID: 316
		private bool _isPosFine = true;

		// Token: 0x0400013D RID: 317
		private XmlElement _BindElement;
	}
}
