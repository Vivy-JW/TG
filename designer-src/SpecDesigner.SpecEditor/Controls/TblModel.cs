using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Input;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor.Controls
{
	// Token: 0x0200001D RID: 29
	public class TblModel : BaseModel
	{
		// Token: 0x060000C6 RID: 198 RVA: 0x00008321 File Offset: 0x00006521
		public TblModel(string name, string parent, XElement xml, PackageKey programKey)
			: base(programKey)
		{
			this._name = name;
			this._parent = parent;
			this._xml = xml;
		}

		// Token: 0x060000C7 RID: 199 RVA: 0x00008340 File Offset: 0x00006540
		private void createSRModels(XElement element)
		{
			if (element == null)
			{
				return;
			}
			foreach (XElement xelement in element.Elements("sr"))
			{
				this._tbls.Add(new SRModel(xelement, base.ProgramKey));
			}
		}

		// Token: 0x17000013 RID: 19
		// (get) Token: 0x060000C8 RID: 200 RVA: 0x0000843C File Offset: 0x0000663C
		public ObservableCollection<BaseModel> Tbls
		{
			get
			{
				this._tbls = new ObservableCollection<BaseModel>();
				if (this._name != string.Empty)
				{
					this.createSRModels((from t in this._xml.Elements("tbl")
						where t.Attribute("name") != null && t.Attribute("name").Value == this._name
						select t).FirstOrDefault<XElement>());
				}
				if (this._parent == null)
				{
					this._tbls.Add(new TblModel("", "", this._xml, base.ProgramKey)
					{
						DisplayName = ""
					});
				}
				else
				{
					IEnumerable<XElement> enumerable = from tbl in this._xml.Elements("tbl")
						where tbl.Attribute("parent").Value == this._name && tbl.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
						select tbl;
					foreach (XElement xelement in enumerable)
					{
						string value = xelement.Attribute("name").Value;
						string text = TableColumnHelper.GetTableDesc(value);
						if (text == null)
						{
							text = value;
						}
						this._tbls.Add(new TblModel(value, this._name, this._xml, base.ProgramKey)
						{
							DisplayName = text
						});
					}
				}
				return this._tbls;
			}
		}

		// Token: 0x17000014 RID: 20
		// (get) Token: 0x060000C9 RID: 201 RVA: 0x000085A8 File Offset: 0x000067A8
		// (set) Token: 0x060000CA RID: 202 RVA: 0x000085B0 File Offset: 0x000067B0
		public string DisplayName
		{
			get
			{
				return this._displayName;
			}
			set
			{
				this._displayName = value;
			}
		}

		// Token: 0x17000015 RID: 21
		// (get) Token: 0x060000CB RID: 203 RVA: 0x000085DC File Offset: 0x000067DC
		// (set) Token: 0x060000CC RID: 204 RVA: 0x00008684 File Offset: 0x00006884
		public string PK
		{
			get
			{
				if (this.Name == base.ProgramKey.Program)
				{
					return "";
				}
				XElement xelement = (from t in this._xml.Elements("tbl")
					where t.Attribute("name").Value == this.Name
					select t).FirstOrDefault<XElement>();
				if (xelement.Attribute("pk") != null)
				{
					return xelement.Attribute("pk").Value;
				}
				return "";
			}
			set
			{
				XElement xelement = (from t in this._xml.Elements("tbl")
					where t.Attribute("name").Value == this.Name
					select t).FirstOrDefault<XElement>();
				xelement.Add(new XAttribute("pk", value));
			}
		}

		// Token: 0x17000016 RID: 22
		// (get) Token: 0x060000CD RID: 205 RVA: 0x000086F8 File Offset: 0x000068F8
		// (set) Token: 0x060000CE RID: 206 RVA: 0x000087A0 File Offset: 0x000069A0
		public string Fk_Target
		{
			get
			{
				if (this.Name == base.ProgramKey.Program)
				{
					return "";
				}
				XElement xelement = (from t in this._xml.Elements("tbl")
					where t.Attribute("name").Value == this.Name
					select t).FirstOrDefault<XElement>();
				if (xelement.Attribute("fk_target") != null)
				{
					return xelement.Attribute("fk_target").Value;
				}
				return "";
			}
			set
			{
				XElement xelement = (from t in this._xml.Elements("tbl")
					where t.Attribute("name").Value == this.Name
					select t).FirstOrDefault<XElement>();
				xelement.Add(new XAttribute("fk_target", value));
			}
		}

		// Token: 0x17000017 RID: 23
		// (get) Token: 0x060000CF RID: 207 RVA: 0x00008814 File Offset: 0x00006A14
		// (set) Token: 0x060000D0 RID: 208 RVA: 0x000088BC File Offset: 0x00006ABC
		public string Fk_Source
		{
			get
			{
				if (this.Name == base.ProgramKey.Program)
				{
					return "";
				}
				XElement xelement = (from t in this._xml.Elements("tbl")
					where t.Attribute("name").Value == this.Name
					select t).FirstOrDefault<XElement>();
				if (xelement.Attribute("fk_source") != null)
				{
					return xelement.Attribute("fk_source").Value;
				}
				return "";
			}
			set
			{
				XElement xelement = (from t in this._xml.Elements("tbl")
					where t.Attribute("name").Value == this.Name
					select t).FirstOrDefault<XElement>();
				xelement.Add(new XAttribute("fk_source", value));
			}
		}

		// Token: 0x17000018 RID: 24
		// (get) Token: 0x060000D1 RID: 209 RVA: 0x0000890B File Offset: 0x00006B0B
		// (set) Token: 0x060000D2 RID: 210 RVA: 0x00008A30 File Offset: 0x00006C30
		public string Name
		{
			get
			{
				if (!(this._name == ""))
				{
					return this._name;
				}
				return base.ProgramKey.Program;
			}
			set
			{
				if (this._name == value)
				{
					return;
				}
				XElement oldNode = (from t in this._xml.Elements("tbl")
					where t.Attribute("name").Value == this._name
					select t).First<XElement>();
				XElement xelement = (from t in this._xml.Elements("tbl")
					where t.Attribute("name").Value == value
					select t).FirstOrDefault<XElement>();
				if (xelement != null)
				{
					xelement.Remove();
				}
				xelement = new XElement(oldNode);
				xelement.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY));
				xelement.SetAttributeValue("name", value);
				oldNode.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE));
				oldNode.Elements("sr").Attributes("status").ToList<XAttribute>()
					.ForEach(delegate(XAttribute s)
					{
						s.Value = ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE);
					});
				(from t in this._xml.Elements("tbl")
					where t.Attribute("parent").Value == oldNode.Attribute("name").Value && t.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select t).ToList<XElement>().ForEach(delegate(XElement t)
				{
					t.SetAttributeValue("parent", value);
					t.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY));
				});
				this._xml.Add(xelement);
				this._name = value;
				string tableDesc = TableColumnHelper.GetTableDesc(value);
				if (tableDesc == null)
				{
					this._displayName = value;
				}
				this._displayName = tableDesc;
				TableSchemaAssociationWindow.RefreshDataSource();
				base.NotifyPropertyChanged("DisplayName");
			}
		}

		// Token: 0x17000019 RID: 25
		// (get) Token: 0x060000D3 RID: 211 RVA: 0x00008C30 File Offset: 0x00006E30
		public bool IsEditable
		{
			get
			{
				XElement xelement = (from t in this._xml.Elements("tbl")
					where t.Attribute("name") != null && this.Name.Equals(t.Attribute("name").Value)
					select t).FirstOrDefault<XElement>();
				return xelement != null && xelement.Attribute("main").Value != "Y";
			}
		}

		// Token: 0x1700001A RID: 26
		// (get) Token: 0x060000D4 RID: 212 RVA: 0x00008CC4 File Offset: 0x00006EC4
		public string Status
		{
			get
			{
				XElement xelement = (from t in this._xml.Elements("tbl")
					where t.Attribute("name") != null && this.Name.Equals(t.Attribute("name").Value)
					select t).FirstOrDefault<XElement>();
				if (xelement != null)
				{
					return xelement.Attribute("status").Value;
				}
				return null;
			}
		}

		// Token: 0x1700001B RID: 27
		// (get) Token: 0x060000D5 RID: 213 RVA: 0x00008D29 File Offset: 0x00006F29
		public ICommand AddCommand
		{
			get
			{
				if (this.addCommand == null)
				{
					this.addCommand = new RelayCommand(delegate(object o)
					{
						this.OnAddNode(this);
					}, (object o) => this.CanAdd(this));
				}
				return this.addCommand;
			}
		}

		// Token: 0x060000D6 RID: 214 RVA: 0x00008D5C File Offset: 0x00006F5C
		public bool CanAdd(TblModel tbl)
		{
			return tbl.Status != ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE);
		}

		// Token: 0x060000D7 RID: 215 RVA: 0x00008D98 File Offset: 0x00006F98
		public void OnAddNode(object nullObject)
		{
			string newName = TblModel.GetNewName("noname", this._xml);
			XElement xelement;
			if (this._name == "")
			{
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo;
				xelement = XElement.Parse("<tbl name='' pk='' fk_source='' fk_target='' parent='' main='N' src='' status=''/>", LoadOptions.None);
				xelement.SetAttributeValue("src", specificationInfo.Env);
			}
			else
			{
				xelement = (from t in this._xml.Elements("tbl")
					where t.Attribute("name").Value == this._name
					select t).First<XElement>();
			}
			XElement xelement2 = new XElement(xelement);
			xelement2.SetAttributeValue("parent", this._name);
			xelement2.SetAttributeValue("name", newName);
			xelement2.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.CREATE));
			xelement2.SetAttributeValue("main", "N");
			xelement2.Elements().Remove<XElement>();
			this._xml.Add(xelement2);
			this._tbls.Add(new TblModel(newName, this.Name, this._xml, base.ProgramKey)
			{
				DisplayName = ""
			});
			base.NotifyPropertyChanged("Tbls");
			TableSchemaAssociationWindow.RefreshDataSource();
		}

		// Token: 0x1700001C RID: 28
		// (get) Token: 0x060000D8 RID: 216 RVA: 0x00008F05 File Offset: 0x00007105
		public ICommand DeleteCommand
		{
			get
			{
				if (this.deleteCommand == null)
				{
					this.deleteCommand = new RelayCommand(delegate(object o)
					{
						this.OnDeleteNode(this);
					}, (object o) => this.CanDelete(this));
				}
				return this.deleteCommand;
			}
		}

		// Token: 0x060000D9 RID: 217 RVA: 0x00008F38 File Offset: 0x00007138
		public bool CanDelete(TblModel obj)
		{
			return obj.IsEditable;
		}

		// Token: 0x060000DA RID: 218 RVA: 0x00008FB4 File Offset: 0x000071B4
		public void OnDeleteNode(object nullObject)
		{
			string text = Application.Current.FindResource("ads_ActionDelConfirm") as string;
			text = string.Format(text, this.Name);
			if (DesignerMessageBox.Show(text, "", MessageBoxButton.YesNo) == MessageBoxResult.No)
			{
				return;
			}
			XElement xelement = (from t in this._xml.Elements("tbl")
				where t.Attribute("name").Value == this.Name && t.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
				select t).First<XElement>();
			xelement.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE));
			xelement.Elements("sr").ToList<XElement>().ForEach(delegate(XElement r)
			{
				r.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE));
			});
			this.deepRemoveByParentName(this.Name);
			base.NotifyPropertyChanged("Tbls");
			TableSchemaAssociationWindow.RefreshDataSource();
			TableSchemaAssociationWindow.Refresh();
		}

		// Token: 0x060000DB RID: 219 RVA: 0x00009110 File Offset: 0x00007310
		private void deepRemoveByParentName(string parentName)
		{
			List<XElement> list = (from t in this._xml.Elements("tbl")
				where t.Attribute("parent").Value == parentName && t.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
				select t).ToList<XElement>();
			foreach (XElement xelement in list)
			{
				xelement.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE));
				xelement.Elements("sr").ToList<XElement>().ForEach(delegate(XElement sr)
				{
					sr.SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE));
				});
				this.deepRemoveByParentName(xelement.Attribute("name").Value);
			}
		}

		// Token: 0x060000DC RID: 220 RVA: 0x00009204 File Offset: 0x00007404
		private static string GetNewName(string namePrefix, XElement _xml)
		{
			List<int> list = new List<int>();
			foreach (XElement xelement in _xml.Elements("tbl"))
			{
				list.Add(TblModel.getSequenceNumber(xelement.Attribute("name").Value, namePrefix));
			}
			if (list.Count == 0)
			{
				return namePrefix.ToString() + "1";
			}
			int[] array = list.OrderBy<int, int>((int x) => x).ToArray<int>();
			int num = -1;
			int num2 = 0;
			for (int i = 1; i < array.Length; i++)
			{
				if (array[i] - array[num2] > 1)
				{
					num = array[num2];
					break;
				}
				num2++;
			}
			if (num == -1)
			{
				num = array.Max();
			}
			return namePrefix + (num + 1);
		}

		// Token: 0x060000DD RID: 221 RVA: 0x00009310 File Offset: 0x00007510
		private static int getSequenceNumber(string name, string namePrefix)
		{
			string text = "(?=" + namePrefix.ToLowerInvariant() + "\\D{0,}(\\d*$))";
			Regex regex = new Regex(text);
			Match match = regex.Match(name.ToLowerInvariant());
			if (match.Success && !"".Equals(match.Groups[1].Value))
			{
				return (int)short.Parse(match.Groups[1].Value);
			}
			return 0;
		}

		// Token: 0x04000083 RID: 131
		private XElement _xml;

		// Token: 0x04000084 RID: 132
		private string _name;

		// Token: 0x04000085 RID: 133
		private string _parent;

		// Token: 0x04000086 RID: 134
		private ObservableCollection<BaseModel> _tbls;

		// Token: 0x04000087 RID: 135
		private string _displayName;

		// Token: 0x04000088 RID: 136
		private RelayCommand addCommand;

		// Token: 0x04000089 RID: 137
		private RelayCommand deleteCommand;
	}
}
