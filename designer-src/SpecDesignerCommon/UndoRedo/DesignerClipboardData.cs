using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Runtime.Serialization;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using SpecDesignerCommon.Views;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x020000AA RID: 170
	[Serializable]
	public class DesignerClipboardData
	{
		// Token: 0x17000208 RID: 520
		// (get) Token: 0x06000725 RID: 1829 RVA: 0x0001FDDC File Offset: 0x0001DFDC
		// (set) Token: 0x06000726 RID: 1830 RVA: 0x0001FDE4 File Offset: 0x0001DFE4
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x17000209 RID: 521
		// (get) Token: 0x06000727 RID: 1831 RVA: 0x0001FDED File Offset: 0x0001DFED
		// (set) Token: 0x06000728 RID: 1832 RVA: 0x0001FDF5 File Offset: 0x0001DFF5
		public List<DesignerClipboardData.StringFormSpecModel> Components { get; private set; }

		// Token: 0x1700020A RID: 522
		// (get) Token: 0x06000729 RID: 1833 RVA: 0x0001FDFE File Offset: 0x0001DFFE
		// (set) Token: 0x0600072A RID: 1834 RVA: 0x0001FE06 File Offset: 0x0001E006
		public int StartIndex { get; private set; }

		// Token: 0x0600072B RID: 1835 RVA: 0x0001FE0F File Offset: 0x0001E00F
		public DesignerClipboardData(PackageKey key, ComponentType containerType)
		{
			this.ProgramKey = key;
			this.Components = new List<DesignerClipboardData.StringFormSpecModel>();
			this._containerType = containerType;
			this.StartIndex = -1;
		}

		// Token: 0x0600072C RID: 1836 RVA: 0x0001FE38 File Offset: 0x0001E038
		public void AppendSelectedItem(XmlElement form)
		{
			if (form == null)
			{
				return;
			}
			FormSpecModel formSpecModel = SettingManager.Get().GetTzpManger(form.Key).SpecificationInfo.FindNodeByName(form.Name);
			DesignerClipboardData.StringFormSpecModel stringFormSpecModel = new DesignerClipboardData.StringFormSpecModel(form.ToString());
			stringFormSpecModel.SpecAction = ((formSpecModel.SpecAction == null) ? null : formSpecModel.SpecAction.ToString());
			stringFormSpecModel.SpecField = ((formSpecModel.SpecField == null) ? null : formSpecModel.SpecField.ToString());
			stringFormSpecModel.SpecHelpCode = ((formSpecModel.SpecHelpCode == null) ? null : formSpecModel.SpecHelpCode.ToString());
			stringFormSpecModel.SpecMultiLang = ((formSpecModel.SpecMultiLang == null) ? null : formSpecModel.SpecMultiLang.ToString());
			stringFormSpecModel.SpecProgRel = ((formSpecModel.SpecProgRel == null) ? null : formSpecModel.SpecProgRel.ToString());
			stringFormSpecModel.SpecReference = ((formSpecModel.SpecReference == null) ? null : formSpecModel.SpecReference.ToString());
			stringFormSpecModel.SpecTree = ((formSpecModel.SpecTree == null) ? null : formSpecModel.SpecTree.ToString());
			stringFormSpecModel.SpecExclude = ((!formSpecModel.IsExcluded) ? null : true.ToString());
			stringFormSpecModel.Index = form.Index;
			this.Components.Add(stringFormSpecModel);
			switch (this._containerType)
			{
			case ComponentType.Table:
			case ComponentType.Tree:
				if (-1 == this.StartIndex || this.StartIndex > form.Index)
				{
					this.StartIndex = form.Index;
				}
				break;
			}
			if (this._localList == null)
			{
				this._localList = new Dictionary<string, DesignerClipboardData.LocalSetting>();
			}
			foreach (DesignerClipboardData.LocalSetting localSetting in DesignerClipboardData.LocaleSettingHelper.GetAllLocaleSetting(this.ProgramKey, form))
			{
				if (!this._localList.ContainsKey(localSetting.Key))
				{
					this._localList.Add(localSetting.Key, localSetting);
				}
			}
		}

		// Token: 0x0600072D RID: 1837 RVA: 0x000201E4 File Offset: 0x0001E3E4
		public IEnumerable<XmlElement> GetForms(PackageKey key)
		{
			foreach (DesignerClipboardData.StringFormSpecModel model in this.Components)
			{
				XmlElement form = XmlElement.Create(key, model.Form);
				yield return form;
			}
			yield break;
		}

		// Token: 0x0600072E RID: 1838 RVA: 0x000205C8 File Offset: 0x0001E7C8
		public IEnumerable<FormSpecModel> GetSelections(PackageKey key)
		{
			SpecificationInfo info = SettingManager.Get().GetTzpManger(key).SpecificationInfo;
			foreach (DesignerClipboardData.StringFormSpecModel model in this.Components)
			{
				XmlElement form = XmlElement.Create(key, model.Form);
				FormSpecModel fsm = new FormSpecModel(key, form);
				if (model.SpecAction != null)
				{
					fsm.SpecAction = SpecActionNode.Create(key, XElement.Parse(model.SpecAction), info.Env);
				}
				if (model.SpecField != null)
				{
					fsm.SpecField = SpecFieldNode.Create(key, XElement.Parse(model.SpecField), info.Env);
				}
				if (model.SpecHelpCode != null)
				{
					fsm.SpecHelpCode = SpecHelpCodeNode.Create(key, XElement.Parse(model.SpecHelpCode), info.Env);
				}
				if (model.SpecMultiLang != null)
				{
					fsm.SpecMultiLang = SpecMultiLangNode.Create(key, XElement.Parse(model.SpecMultiLang), info.Env);
				}
				if (model.SpecProgRel != null)
				{
					fsm.SpecProgRel = SpecProgRelNode.Create(key, XElement.Parse(model.SpecProgRel), info.Env);
				}
				if (model.SpecReference != null)
				{
					fsm.SpecReference = SpecReferenceNode.Create(key, XElement.Parse(model.SpecReference), info.Env);
				}
				if (model.SpecTree != null)
				{
					fsm.SpecTree = SpecTreeNode.Create(key, XElement.Parse(model.SpecTree));
				}
				if (model.SpecExclude != null)
				{
					fsm.IsExcluded = true;
				}
				yield return fsm;
			}
			yield break;
		}

		// Token: 0x0600072F RID: 1839 RVA: 0x000205EC File Offset: 0x0001E7EC
		public void ImportlocalItems(PackageKey key)
		{
			if (this._localList.Count == 0)
			{
				return;
			}
			List<DesignerClipboardData.LocalSetting> list = new List<DesignerClipboardData.LocalSetting>();
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(key).SpecificationInfo;
			foreach (DesignerClipboardData.LocalSetting localSetting in this._localList.Values)
			{
				string text = (localSetting.IsAct ? specificationInfo.GetActLocalStringText(localSetting.Key) : specificationInfo.GetFieldLocalStringText(localSetting.Key));
				if (text != null)
				{
					if (text == localSetting.Value)
					{
						continue;
					}
					localSetting.IsDuplicated = true;
				}
				list.Add(localSetting);
			}
			if (list.Count == 0)
			{
				return;
			}
			new LocalItemsSelectionWindow
			{
				DataContext = list
			}.ShowDialog();
			foreach (DesignerClipboardData.LocalSetting localSetting2 in list)
			{
				if (localSetting2.IsImport)
				{
					if (localSetting2.IsAct)
					{
						specificationInfo.SetActLocalStringText(localSetting2.Key, localSetting2.Value);
					}
					else
					{
						specificationInfo.SetFieldLocalStringText(localSetting2.Key, localSetting2.Value);
					}
				}
			}
		}

		// Token: 0x06000730 RID: 1840 RVA: 0x0002073C File Offset: 0x0001E93C
		public bool CanExecutePaste(XmlElement newContainer)
		{
			bool flag = true;
			ComponentType containerType = this._containerType;
			if (containerType != ComponentType.Unknown)
			{
				switch (containerType)
				{
				case ComponentType.Table:
				case ComponentType.Tree:
					flag = newContainer.Type == this._containerType;
					break;
				}
			}
			else
			{
				flag = false;
			}
			if (flag)
			{
				ComponentType type = newContainer.Type;
				if (type != ComponentType.Unknown)
				{
					switch (type)
					{
					case ComponentType.Table:
					case ComponentType.Tree:
						flag = newContainer.Type == this._containerType;
						break;
					}
				}
				else
				{
					flag = false;
				}
			}
			if (flag)
			{
				foreach (XmlElement xmlElement in this.GetForms(newContainer.Key))
				{
					if (!ComponentFactory.AcceptMimes(newContainer, xmlElement.Type))
					{
						flag = false;
						break;
					}
				}
			}
			return flag;
		}

		// Token: 0x0400028D RID: 653
		private ComponentType _containerType;

		// Token: 0x0400028E RID: 654
		private Dictionary<string, DesignerClipboardData.LocalSetting> _localList;

		// Token: 0x020000AB RID: 171
		[Serializable]
		public class StringFormSpecModel
		{
			// Token: 0x1700020B RID: 523
			// (get) Token: 0x06000731 RID: 1841 RVA: 0x0002080C File Offset: 0x0001EA0C
			// (set) Token: 0x06000732 RID: 1842 RVA: 0x00020814 File Offset: 0x0001EA14
			public string Form { get; set; }

			// Token: 0x1700020C RID: 524
			// (get) Token: 0x06000733 RID: 1843 RVA: 0x0002081D File Offset: 0x0001EA1D
			// (set) Token: 0x06000734 RID: 1844 RVA: 0x00020825 File Offset: 0x0001EA25
			public string SpecField { get; set; }

			// Token: 0x1700020D RID: 525
			// (get) Token: 0x06000735 RID: 1845 RVA: 0x0002082E File Offset: 0x0001EA2E
			// (set) Token: 0x06000736 RID: 1846 RVA: 0x00020836 File Offset: 0x0001EA36
			public string SpecAction { get; set; }

			// Token: 0x1700020E RID: 526
			// (get) Token: 0x06000737 RID: 1847 RVA: 0x0002083F File Offset: 0x0001EA3F
			// (set) Token: 0x06000738 RID: 1848 RVA: 0x00020847 File Offset: 0x0001EA47
			public string SpecHelpCode { get; set; }

			// Token: 0x1700020F RID: 527
			// (get) Token: 0x06000739 RID: 1849 RVA: 0x00020850 File Offset: 0x0001EA50
			// (set) Token: 0x0600073A RID: 1850 RVA: 0x00020858 File Offset: 0x0001EA58
			public string SpecMultiLang { get; set; }

			// Token: 0x17000210 RID: 528
			// (get) Token: 0x0600073B RID: 1851 RVA: 0x00020861 File Offset: 0x0001EA61
			// (set) Token: 0x0600073C RID: 1852 RVA: 0x00020869 File Offset: 0x0001EA69
			public string SpecProgRel { get; set; }

			// Token: 0x17000211 RID: 529
			// (get) Token: 0x0600073D RID: 1853 RVA: 0x00020872 File Offset: 0x0001EA72
			// (set) Token: 0x0600073E RID: 1854 RVA: 0x0002087A File Offset: 0x0001EA7A
			public string SpecReference { get; set; }

			// Token: 0x17000212 RID: 530
			// (get) Token: 0x0600073F RID: 1855 RVA: 0x00020883 File Offset: 0x0001EA83
			// (set) Token: 0x06000740 RID: 1856 RVA: 0x0002088B File Offset: 0x0001EA8B
			public string SpecTree { get; set; }

			// Token: 0x17000213 RID: 531
			// (get) Token: 0x06000741 RID: 1857 RVA: 0x00020894 File Offset: 0x0001EA94
			// (set) Token: 0x06000742 RID: 1858 RVA: 0x0002089C File Offset: 0x0001EA9C
			public string SpecExclude { get; set; }

			// Token: 0x17000214 RID: 532
			// (get) Token: 0x06000743 RID: 1859 RVA: 0x000208A5 File Offset: 0x0001EAA5
			// (set) Token: 0x06000744 RID: 1860 RVA: 0x000208AD File Offset: 0x0001EAAD
			public int Index { get; set; }

			// Token: 0x06000745 RID: 1861 RVA: 0x000208B8 File Offset: 0x0001EAB8
			public StringFormSpecModel()
			{
				this.Form = null;
				this.SpecField = null;
				this.SpecAction = null;
				this.SpecHelpCode = null;
				this.SpecMultiLang = null;
				this.SpecProgRel = null;
				this.SpecReference = null;
				this.SpecTree = null;
				this.SpecExclude = null;
			}

			// Token: 0x06000746 RID: 1862 RVA: 0x0002090A File Offset: 0x0001EB0A
			public StringFormSpecModel(string form)
				: this()
			{
				this.Form = form.ToString();
			}
		}

		// Token: 0x020000AC RID: 172
		private static class LocaleSettingHelper
		{
			// Token: 0x06000747 RID: 1863 RVA: 0x00020DBC File Offset: 0x0001EFBC
			public static IEnumerable<DesignerClipboardData.LocalSetting> GetAllLocaleSetting(PackageKey packageKey, XmlElement form)
			{
				SpecificationInfo info = SettingManager.Get().GetTzpManger(packageKey).SpecificationInfo;
				string localString = null;
				string text = form.GetAttribute("text");
				if (text != null && text.Length > 0)
				{
					ComponentType type = form.Type;
					if (type == ComponentType.RadioGroupItem)
					{
						localString = info.GetItemLocalStringText(text);
					}
					else
					{
						localString = info.GetFieldLocalStringText(text);
					}
					if (localString != null)
					{
						yield return new DesignerClipboardData.LocalSetting(text, localString);
					}
				}
				string title = form.GetAttribute("title");
				if (title != null && title.Length > 0)
				{
					localString = info.GetFieldLocalStringText(title);
					if (localString != null)
					{
						yield return new DesignerClipboardData.LocalSetting(title, localString);
					}
				}
				string comment = form.GetAttribute("comment");
				if (comment != null && comment.Length > 0)
				{
					ComponentType type2 = form.Type;
					if (type2 == ComponentType.Button)
					{
						localString = info.GetActLocalStringText(comment);
						if (localString != null)
						{
							yield return new DesignerClipboardData.LocalSetting(comment, localString)
							{
								IsAct = true
							};
						}
					}
					else
					{
						localString = info.GetFieldLocalStringText(comment);
						if (localString != null)
						{
							yield return new DesignerClipboardData.LocalSetting(comment, localString);
						}
					}
				}
				if (form.HasItems)
				{
					foreach (XmlElement child in form.Items)
					{
						foreach (DesignerClipboardData.LocalSetting s in DesignerClipboardData.LocaleSettingHelper.GetAllLocaleSetting(packageKey, child))
						{
							yield return s;
						}
					}
				}
				yield break;
			}
		}

		// Token: 0x020000AD RID: 173
		[DataContract]
		[Serializable]
		private class LocalSetting : INotifyPropertyChanged
		{
			// Token: 0x17000215 RID: 533
			// (get) Token: 0x06000748 RID: 1864 RVA: 0x00020DE0 File Offset: 0x0001EFE0
			// (set) Token: 0x06000749 RID: 1865 RVA: 0x00020DE8 File Offset: 0x0001EFE8
			[DataMember]
			public string Key { get; private set; }

			// Token: 0x17000216 RID: 534
			// (get) Token: 0x0600074A RID: 1866 RVA: 0x00020DF1 File Offset: 0x0001EFF1
			// (set) Token: 0x0600074B RID: 1867 RVA: 0x00020DF9 File Offset: 0x0001EFF9
			[DataMember]
			public string Value { get; private set; }

			// Token: 0x17000217 RID: 535
			// (get) Token: 0x0600074C RID: 1868 RVA: 0x00020E02 File Offset: 0x0001F002
			// (set) Token: 0x0600074D RID: 1869 RVA: 0x00020E0A File Offset: 0x0001F00A
			[DataMember]
			public bool IsAct { get; set; }

			// Token: 0x17000218 RID: 536
			// (get) Token: 0x0600074E RID: 1870 RVA: 0x00020E13 File Offset: 0x0001F013
			// (set) Token: 0x0600074F RID: 1871 RVA: 0x00020E1B File Offset: 0x0001F01B
			public bool IsImport
			{
				get
				{
					return this._isImport;
				}
				set
				{
					this._isImport = value;
					this.RaisePropertyChanged("IsImport");
				}
			}

			// Token: 0x17000219 RID: 537
			// (get) Token: 0x06000750 RID: 1872 RVA: 0x00020E2F File Offset: 0x0001F02F
			// (set) Token: 0x06000751 RID: 1873 RVA: 0x00020E37 File Offset: 0x0001F037
			public bool IsDuplicated
			{
				get
				{
					return this._isDuplicated;
				}
				set
				{
					this._isDuplicated = value;
					if (this._isDuplicated)
					{
						this.IsImport = false;
					}
				}
			}

			// Token: 0x06000752 RID: 1874 RVA: 0x00020E4F File Offset: 0x0001F04F
			public LocalSetting(string key, string value)
			{
				this.Key = key;
				this.Value = value;
				this.IsAct = false;
				this.IsDuplicated = false;
			}

			// Token: 0x06000753 RID: 1875 RVA: 0x00020E7C File Offset: 0x0001F07C
			public override bool Equals(object obj)
			{
				if (!(obj is string))
				{
					return false;
				}
				DesignerClipboardData.LocalSetting localSetting = (DesignerClipboardData.LocalSetting)obj;
				return this.Key == localSetting.Key && this.Value == localSetting.Value && this.IsAct == localSetting.IsAct;
			}

			// Token: 0x06000754 RID: 1876 RVA: 0x00020ED0 File Offset: 0x0001F0D0
			public static bool operator ==(DesignerClipboardData.LocalSetting x, DesignerClipboardData.LocalSetting y)
			{
				if (object.ReferenceEquals(x, null))
				{
					return object.ReferenceEquals(y, null);
				}
				return x.Equals(y);
			}

			// Token: 0x06000755 RID: 1877 RVA: 0x00020EEA File Offset: 0x0001F0EA
			public static bool operator !=(DesignerClipboardData.LocalSetting x, DesignerClipboardData.LocalSetting y)
			{
				return !(x == y);
			}

			// Token: 0x1400001A RID: 26
			// (add) Token: 0x06000756 RID: 1878 RVA: 0x00020EF8 File Offset: 0x0001F0F8
			// (remove) Token: 0x06000757 RID: 1879 RVA: 0x00020F30 File Offset: 0x0001F130
			public event PropertyChangedEventHandler PropertyChanged;

			// Token: 0x06000758 RID: 1880 RVA: 0x00020F65 File Offset: 0x0001F165
			private void RaisePropertyChanged(string propertyName)
			{
				if (this.PropertyChanged != null)
				{
					this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
				}
			}

			// Token: 0x0400029C RID: 668
			private bool _isImport = true;

			// Token: 0x0400029D RID: 669
			private bool _isDuplicated;

			// Token: 0x020000AE RID: 174
			public class LocalSettingComparer : IEqualityComparer<DesignerClipboardData.LocalSetting>
			{
				// Token: 0x06000759 RID: 1881 RVA: 0x00020F81 File Offset: 0x0001F181
				public bool Equals(DesignerClipboardData.LocalSetting x, DesignerClipboardData.LocalSetting y)
				{
					return x.Key == y.Key && x.Value == y.Value;
				}

				// Token: 0x0600075A RID: 1882 RVA: 0x00020FAC File Offset: 0x0001F1AC
				public int GetHashCode(DesignerClipboardData.LocalSetting obj)
				{
					return obj.Key.GetHashCode() ^ obj.Value.GetHashCode() ^ obj.IsAct.GetHashCode();
				}
			}
		}
	}
}
