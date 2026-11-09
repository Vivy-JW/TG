using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.IO;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Xml;
using System.Xml.Linq;
using System.Xml.Schema;
using System.Xml.Serialization;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;
using SpecDesignerCustomException;

namespace SpecDesignerCommon
{
	// Token: 0x02000100 RID: 256
	public class SpecificationInfo : INotifyPropertyChanged
	{
		// Token: 0x0600084A RID: 2122 RVA: 0x0002454A File Offset: 0x0002274A
		public static SpecificationInfo Create(TzpManager tzpManager)
		{
			if (string.IsNullOrEmpty(tzpManager.Tsd) || string.IsNullOrEmpty(tzpManager.GeneroFormString))
			{
				return null;
			}
			return new SpecificationInfo(tzpManager);
		}

		// Token: 0x1700024B RID: 587
		// (get) Token: 0x0600084B RID: 2123 RVA: 0x0002456E File Offset: 0x0002276E
		public FormSpecDictionary FormSpeDictionary
		{
			get
			{
				return this._formSpecDic;
			}
		}

		// Token: 0x1700024C RID: 588
		// (get) Token: 0x0600084C RID: 2124 RVA: 0x00024576 File Offset: 0x00022776
		// (set) Token: 0x0600084D RID: 2125 RVA: 0x0002457E File Offset: 0x0002277E
		public PackageKey Key { get; private set; }

		// Token: 0x1700024D RID: 589
		// (get) Token: 0x0600084E RID: 2126 RVA: 0x00024587 File Offset: 0x00022787
		// (set) Token: 0x0600084F RID: 2127 RVA: 0x0002458F File Offset: 0x0002278F
		public XElement TSDElement { get; private set; }

		// Token: 0x1700024E RID: 590
		// (get) Token: 0x06000850 RID: 2128 RVA: 0x00024598 File Offset: 0x00022798
		// (set) Token: 0x06000851 RID: 2129 RVA: 0x000245A0 File Offset: 0x000227A0
		public XElement TSD2Element { get; private set; }

		// Token: 0x1700024F RID: 591
		// (get) Token: 0x06000852 RID: 2130 RVA: 0x000245A9 File Offset: 0x000227A9
		// (set) Token: 0x06000853 RID: 2131 RVA: 0x000245B1 File Offset: 0x000227B1
		public XElement FormElement { get; private set; }

		// Token: 0x17000250 RID: 592
		// (get) Token: 0x06000854 RID: 2132 RVA: 0x000245BA File Offset: 0x000227BA
		// (set) Token: 0x06000855 RID: 2133 RVA: 0x000245C2 File Offset: 0x000227C2
		public SpecDesignerCommon.ViewModel.XmlElement FormNode { get; private set; }

		// Token: 0x17000251 RID: 593
		// (get) Token: 0x06000856 RID: 2134 RVA: 0x000245CB File Offset: 0x000227CB
		// (set) Token: 0x06000857 RID: 2135 RVA: 0x000245D3 File Offset: 0x000227D3
		public XElement CiteSTD { get; private set; }

		// Token: 0x17000252 RID: 594
		// (get) Token: 0x06000858 RID: 2136 RVA: 0x000245DC File Offset: 0x000227DC
		public string Ver
		{
			get
			{
				if (this.TSDElement != null && this.TSDElement.Attribute("ver") != null)
				{
					return this.TSDElement.Attribute("ver").Value;
				}
				return string.Empty;
			}
		}

		// Token: 0x17000253 RID: 595
		// (get) Token: 0x06000859 RID: 2137 RVA: 0x00024628 File Offset: 0x00022828
		public string Env
		{
			get
			{
				if (this.TSDElement != null && this.TSDElement.Attribute("env") != null)
				{
					return this.TSDElement.Attribute("env").Value;
				}
				return string.Empty;
			}
		}

		// Token: 0x17000254 RID: 596
		// (get) Token: 0x0600085A RID: 2138 RVA: 0x00024674 File Offset: 0x00022874
		// (set) Token: 0x0600085B RID: 2139 RVA: 0x0002467C File Offset: 0x0002287C
		public TableAssociationModel AssociateTable { get; set; }

		// Token: 0x17000255 RID: 597
		// (get) Token: 0x0600085C RID: 2140 RVA: 0x00024685 File Offset: 0x00022885
		// (set) Token: 0x0600085D RID: 2141 RVA: 0x0002468D File Offset: 0x0002288D
		public DatabaseSourceViewModel DatabaseSource { get; set; }

		// Token: 0x17000256 RID: 598
		// (get) Token: 0x0600085E RID: 2142 RVA: 0x00024696 File Offset: 0x00022896
		// (set) Token: 0x0600085F RID: 2143 RVA: 0x0002469E File Offset: 0x0002289E
		public bool IsTsdValidated { get; private set; }

		// Token: 0x17000257 RID: 599
		// (get) Token: 0x06000860 RID: 2144 RVA: 0x000246A7 File Offset: 0x000228A7
		// (set) Token: 0x06000861 RID: 2145 RVA: 0x000246AF File Offset: 0x000228AF
		public bool IsUIValidated { get; private set; }

		// Token: 0x06000862 RID: 2146 RVA: 0x000246B8 File Offset: 0x000228B8
		private SpecificationInfo(TzpManager tzpManager)
		{
			if (SettingManager.Get().Info_ModFd == null)
			{
				DesignerMessageBox.Show("mod-fd.spec " + Application.Current.FindResource("Message_LoadingFailed"));
				throw new FileNotFoundException("mod-fd.spec");
			}
			if (string.IsNullOrEmpty(SettingManager.Get().Info_CoreBr))
			{
				DesignerMessageBox.Show("core-br.spec " + Application.Current.FindResource("Message_LoadingFailed"));
				throw new FileNotFoundException("core-br.spec");
			}
			string content = SettingManager.Get().Info_ModFd.Content;
			string info_CoreBr = SettingManager.Get().Info_CoreBr;
			ComponentFactory.SetSpecification(info_CoreBr, new ModFdInfo(content));
			this._tzpManager = tzpManager;
			this.Key = this._tzpManager.ProgramKey;
			this.TSDElement = XElement.Parse(tzpManager.Tsd);
			this.FormElement = XElement.Parse(this._tzpManager.GeneroFormString);
			this._screenRecordManager = new ScreenRecordManager(this.FormElement);
			this.ConvertTSDToModel();
			this.AssociateTable = new TableAssociationModel(this.Key, this.TSDElement.Element("table"));
			this.DatabaseSource = new DatabaseSourceViewModel(this.Key, this.AssociateTable, this.Env, this.FormSpeDictionary);
			this.IsTsdValidated = true;
			this.IsUIValidated = true;
			EventAggregatorManager.Global.GetEvent<SaveSettingEvent>().Subscribe(new Action<PackageKey>(this.SaveSpecificationInfo));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.OnTzpFileClose));
			if (this.SpecBinding == null)
			{
				this.ScanSpecBinding();
				return;
			}
			this.LoadSpecBinding();
		}

		// Token: 0x06000863 RID: 2147 RVA: 0x000248CC File Offset: 0x00022ACC
		private void LoadSpecBinding()
		{
			List<TBinding> list = new List<TBinding>();
			foreach (TBinding tbinding in this.SpecBinding)
			{
				FormSpecModel formSpecModel2;
				FormSpecModel formSpecModel = (formSpecModel2 = null);
				this.FormSpeDictionary.TryGetValue(tbinding.ObjectName1.Text, out formSpecModel2);
				this.FormSpeDictionary.TryGetValue(tbinding.ObjectName2.Text, out formSpecModel);
				if (formSpecModel2 == null || formSpecModel == null)
				{
					list.Add(tbinding);
				}
				else
				{
					formSpecModel2.GeneroComponent.BindElement = formSpecModel.GeneroComponent;
					formSpecModel.GeneroComponent.BindElement = formSpecModel2.GeneroComponent;
				}
			}
			foreach (TBinding tbinding2 in list)
			{
				this.SpecBinding.Remove(tbinding2);
			}
		}

		// Token: 0x06000864 RID: 2148 RVA: 0x000249CC File Offset: 0x00022BCC
		private void ScanSpecBinding()
		{
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair in this.FormSpeDictionary)
			{
				FormSpecModel value = keyValuePair.Value;
				if (SpecificationInfo.isValidBindingType(value.Type) && value.SpecField != null && keyValuePair.Key == value.SpecField.Table + "." + value.SpecField.Column)
				{
					string column = value.SpecField.Column;
					FormSpecModel formSpecModel = this.FindNodeByName("lbl_" + column);
					if (formSpecModel != null)
					{
						value.GeneroComponent.BindElement = formSpecModel.GeneroComponent;
						formSpecModel.GeneroComponent.BindElement = value.GeneroComponent;
						this.AddSpecBinding(formSpecModel.GeneroComponent, value.GeneroComponent);
					}
				}
			}
		}

		// Token: 0x06000865 RID: 2149 RVA: 0x00024AC8 File Offset: 0x00022CC8
		public void AddSpecBinding(SpecDesignerCommon.ViewModel.XmlElement e1, SpecDesignerCommon.ViewModel.XmlElement e2)
		{
			if (e1 != null && e2 != null)
			{
				FJSObject fjsobject = new FJSObject
				{
					Text = e1.Name,
					Type = e1.Type
				};
				FJSObject fjsobject2 = new FJSObject
				{
					Text = e2.Name,
					Type = e2.Type
				};
				if (this.SpecBinding == null)
				{
					this.SpecBinding = new List<TBinding>();
				}
				TBinding tbinding = new TBinding
				{
					ID = this.SpecBinding.Count + 1,
					ObjectName1 = fjsobject,
					ObjectName2 = fjsobject2
				};
				if (!this.SpecBinding.Contains(tbinding))
				{
					this.SpecBinding.Add(tbinding);
				}
			}
		}

		// Token: 0x06000866 RID: 2150 RVA: 0x00024B80 File Offset: 0x00022D80
		public static bool isValidBindingType(ComponentType componentType)
		{
			return componentType == ComponentType.ButtonEdit || componentType == ComponentType.ComboBox || componentType == ComponentType.DateEdit || componentType == ComponentType.DateTimeEdit || componentType == ComponentType.Edit || componentType == ComponentType.TextEdit || componentType == ComponentType.TimeEdit || componentType == ComponentType.SpinEdit;
		}

		// Token: 0x06000867 RID: 2151 RVA: 0x00024BE4 File Offset: 0x00022DE4
		private void SaveSpecificationInfo(PackageKey key)
		{
			if (key == this.Key || null == key)
			{
				this.SaveToTSD();
				this.SaveToForm();
				this.SaveBinding();
				this.InitTSDValidateWorker();
				this.InitFormValidateWorker();
				IEnumerable<XElement> enumerable = from p in this.FormElement.Descendants("RecordField")
					where p.Attribute("colAliasName") != null && p.Attribute("colAliasName").Value == string.Empty
					select p;
				foreach (XElement xelement in enumerable)
				{
					xelement.Attribute("colAliasName").Remove();
				}
				SettingManager.Get().GetTzpManger(this.Key).SaveFormFile(this.FormElement.ToString());
				SettingManager.Get().GetTzpManger(this.Key).SaveSpecificationInfo(this.TSDElement.ToString());
				SettingManager.Get().GetTzpManger(this.Key).SaveUpdatedSpecificationInfo(this.TSD2Element.ToString());
			}
		}

		// Token: 0x06000868 RID: 2152 RVA: 0x00024D0C File Offset: 0x00022F0C
		private void OnTzpFileClose(PackageKey key)
		{
			if (!this.Key.Equals(key))
			{
				return;
			}
			this._acts.Clear();
			this.prog_rels.Clear();
			this.ref_fields.Clear();
			this.multi_langs.Clear();
			this.help_codes.Clear();
			this.fields.Clear();
			this.trees.Clear();
			this._formSpecDic.Clear();
			if (this._TSDValidater != null)
			{
				if (this._TSDValidater.IsBusy)
				{
					this._TSDValidater.CancelAsync();
				}
				this._TSDValidater.Dispose();
			}
			EventAggregatorManager.Global.GetEvent<SaveSettingEvent>().Unsubscribe(new Action<PackageKey>(this.SaveSpecificationInfo));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Unsubscribe(new Action<PackageKey>(this.OnTzpFileClose));
		}

		// Token: 0x06000869 RID: 2153 RVA: 0x00024DE4 File Offset: 0x00022FE4
		private void InitTSDValidateWorker()
		{
			if (this._TSDValidater == null)
			{
				this._TSDValidater = new BackgroundWorker();
				this._TSDValidater.WorkerSupportsCancellation = true;
			}
			if (!this._TSDValidater.IsBusy)
			{
				this._TSDValidater.DoWork += this.StartValidateTSD;
				this._TSDValidater.RunWorkerAsync(this.TSDElement.ToString());
			}
		}

		// Token: 0x0600086A RID: 2154 RVA: 0x00024E4C File Offset: 0x0002304C
		private void StartValidateTSD(object sender, DoWorkEventArgs e)
		{
			this._TSDValidater.DoWork -= this.StartValidateTSD;
			if (this._TSDValidater != null && this._TSDValidater.IsBusy && this._TSDValidater.CancellationPending)
			{
				return;
			}
			this.ValidateTsd(e.Argument as string);
		}

		// Token: 0x0600086B RID: 2155 RVA: 0x00025120 File Offset: 0x00023320
		private void ValidateTsd(string tsd)
		{
			string info_TsdXsd = SettingManager.Get().Info_TsdXsd;
			if (info_TsdXsd == null)
			{
				return;
			}
			bool hasError = false;
			XmlTextReader xmlTextReader = new XmlTextReader(new StringReader(info_TsdXsd));
			XDocument xdocument = XDocument.Parse(tsd, LoadOptions.SetLineInfo);
			XmlSchemaSet xmlSchemaSet = new XmlSchemaSet();
			xmlSchemaSet.Add(string.Empty, xmlTextReader);
			xmlSchemaSet.Compile();
			string name;
			string message;
			xdocument.Validate(xmlSchemaSet, delegate(object o, ValidationEventArgs e)
			{
				if (this._TSDValidater != null && this._TSDValidater.IsBusy && this._TSDValidater.CancellationPending)
				{
					return;
				}
				XElement xelement7 = null;
				message = e.Message;
				if (o is XNode)
				{
					xelement7 = o as XElement;
				}
				else if (o is XAttribute)
				{
					xelement7 = (o as XAttribute).Parent;
					if (xelement7.Attribute("status") != null && xelement7.Attribute("status").Value == ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE))
					{
						return;
					}
				}
				if (o is XAttribute)
				{
					if (message.Contains("noEmpty"))
					{
						message = string.Format("'{0}' {1}", (o as XAttribute).Name, Application.Current.FindResource("Message_CantNull") as string);
					}
					DocumentErrorsEventArgs e4 = new DocumentErrorsEventArgs();
					IXmlLineInfo xmlLineInfo = o as IXmlLineInfo;
					name = ((xelement7.Attribute("name") == null) ? "" : xelement7.Attribute("name").Value);
					if (name == "")
					{
						name = ((xelement7.Attribute("id") == null) ? "" : xelement7.Attribute("id").Value);
					}
					e4.ProgramKey = this.Key;
					e4.SourceType = this.Key.PackType;
					e4.ErrorType = ErrorsType.ERROR;
					e4.Time = DateTime.Now;
					e4.Key = name;
					e4.Description = string.Concat(new object[] { name, " : ", xmlLineInfo.LineNumber, " - ", message });
					hasError = true;
					EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e4);
				}
			}, true);
			this.IsTsdValidated = !hasError;
			if (this._TSDValidater != null && this._TSDValidater.IsBusy && this._TSDValidater.CancellationPending)
			{
				return;
			}
			foreach (XElement xelement in xdocument.Descendants("tree"))
			{
				foreach (XElement xelement2 in xelement.Descendants())
				{
					if (this._TSDValidater != null && this._TSDValidater.IsBusy && this._TSDValidater.CancellationPending)
					{
						return;
					}
					string localName;
					switch (localName = xelement2.Name.LocalName)
					{
					case "type":
					case "type2":
					case "type3":
					case "type4":
					case "type5":
					case "type6":
					case "id":
					case "pid":
					case "desc":
					{
						string tableName = xelement2.Attribute("table").Value;
						if (!string.IsNullOrEmpty(tableName) || !string.IsNullOrEmpty(xelement2.Attribute("col").Value))
						{
							if ((from table in TableColumnHelper.GetTables()
								where table.Attribute("name").Value == tableName
								select table).ElementAtOrDefault<XElement>(0) == null)
							{
								message = string.Format("Table：{0} {1}", tableName, Application.Current.FindResource("Message_NotExist") as string);
								DocumentErrorsEventArgs e3 = new DocumentErrorsEventArgs();
								name = ((xelement.Attribute("name") == null) ? "" : xelement.Attribute("name").Value);
								e3.ProgramKey = this.Key;
								e3.SourceType = this.Key.PackType;
								e3.ErrorType = ErrorsType.WARNING;
								e3.Time = DateTime.Now;
								e3.Key = name;
								e3.Description = string.Format("{0} - {1}", name, message);
								EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e3);
							}
							else
							{
								string colName = xelement2.Attribute("col").Value;
								if ((from col in TableColumnHelper.GetColFields(tableName)
									where col.Attribute("name").Value == colName
									select col).ElementAtOrDefault<XElement>(0) == null)
								{
									message = string.Format("Column：{0} {1}", colName, Application.Current.FindResource("Message_NotExist") as string);
									DocumentErrorsEventArgs e2 = new DocumentErrorsEventArgs();
									name = ((xelement.Attribute("name") == null) ? "" : xelement.Attribute("name").Value);
									e2.ProgramKey = this.Key;
									e2.SourceType = this.Key.PackType;
									e2.ErrorType = ErrorsType.WARNING;
									e2.Time = DateTime.Now;
									e2.Key = name;
									e2.Description = string.Format("{0} - {1}", name, message);
									EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e2);
								}
							}
						}
						break;
					}
					}
				}
			}
			if (this._TSDValidater == null || !this._TSDValidater.IsBusy || !this._TSDValidater.CancellationPending)
			{
				foreach (XElement xelement3 in xdocument.Descendants("ref_field"))
				{
					foreach (XElement xelement4 in xelement3.Descendants())
					{
						if (this._TSDValidater != null && this._TSDValidater.IsBusy && this._TSDValidater.CancellationPending)
						{
							return;
						}
						string localName2;
						if ((localName2 = xelement4.Name.LocalName) != null && localName2 == "rfield")
						{
							this.ColumnSettingCheck(xelement4, "correspon_key");
							this.ColumnSettingCheck(xelement4, "ref_fk");
							this.ColumnSettingCheck(xelement4, "ref_dlang");
							this.ColumnSettingCheck(xelement4, "ref_rtn");
						}
					}
				}
				if (this._TSDValidater != null && this._TSDValidater.IsBusy && this._TSDValidater.CancellationPending)
				{
					return;
				}
				foreach (XElement xelement5 in xdocument.Descendants("help_code"))
				{
					foreach (XElement xelement6 in xelement5.Descendants())
					{
						if (this._TSDValidater != null && this._TSDValidater.IsBusy && this._TSDValidater.CancellationPending)
						{
							return;
						}
						string localName3;
						if ((localName3 = xelement6.Name.LocalName) != null && localName3 == "hfield")
						{
							this.ColumnSettingCheck(xelement6, "help_find");
							this.ColumnSettingCheck(xelement6, "help_dlang");
							this.ColumnSettingCheck(xelement6, "help_field");
						}
					}
				}
				return;
			}
		}

		// Token: 0x0600086C RID: 2156 RVA: 0x00025884 File Offset: 0x00023A84
		private void ColumnSettingCheck(XElement child, string attributeName)
		{
			string[] array = child.Attribute(attributeName).Value.Split(new char[] { ',' });
			foreach (string text in array)
			{
				if (!string.IsNullOrEmpty(text) && !TableColumnHelper.CheckColumnExist(text))
				{
					string text2 = string.Format("Column：{0} {1}", text, Application.Current.FindResource("Message_NotExist") as string);
					DocumentErrorsEventArgs e = new DocumentErrorsEventArgs();
					string value = child.Attribute("name").Value;
					e.ProgramKey = this.Key;
					e.SourceType = this.Key.PackType;
					e.ErrorType = ErrorsType.WARNING;
					e.Time = DateTime.Now;
					e.Key = value;
					e.Description = string.Format("{0} - {1}", value, text2);
					EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e);
				}
			}
		}

		// Token: 0x0600086D RID: 2157 RVA: 0x00025994 File Offset: 0x00023B94
		private void InitFormValidateWorker()
		{
			if (this._FormValidater == null)
			{
				this._FormValidater = new BackgroundWorker();
				this._FormValidater.WorkerSupportsCancellation = true;
			}
			if (!this._FormValidater.IsBusy)
			{
				this._FormValidater.DoWork += this.StartValidateForm;
				this._FormValidater.RunWorkerAsync(this.FormElement);
			}
		}

		// Token: 0x0600086E RID: 2158 RVA: 0x000259F8 File Offset: 0x00023BF8
		private void StartValidateForm(object sender, DoWorkEventArgs e)
		{
			this._FormValidater.DoWork -= this.StartValidateTSD;
			if (this._FormValidater != null && this._FormValidater.IsBusy && this._FormValidater.CancellationPending)
			{
				return;
			}
			this.validateForm(e.Argument as XElement);
		}

		// Token: 0x0600086F RID: 2159 RVA: 0x00025AC0 File Offset: 0x00023CC0
		private void validateForm(XElement xml)
		{
			IEnumerable<XElement> enumerable = xml.Element("Form").Elements();
			bool flag = false;
			foreach (XElement xelement in enumerable)
			{
				if (this._TSDValidater != null && this._TSDValidater.IsBusy && this._TSDValidater.CancellationPending)
				{
					return;
				}
				if (xelement.Attribute("tag") == null)
				{
					flag = false;
					break;
				}
				flag = xelement.Attribute("tag").Value.Contains("FormRoot");
			}
			if (!flag)
			{
				DocumentErrorsEventArgs e = new DocumentErrorsEventArgs();
				e.ProgramKey = this.Key;
				e.SourceType = this.Key.PackType;
				e.ErrorType = ErrorsType.INFORMATION;
				e.Time = DateTime.Now;
				e.Key = this.Key.Program;
				e.Description = "can't found template root";
				EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e);
			}
			bool flag2 = false;
			string text = Application.Current.FindResource("Message_DuplicateFieldName") as string;
			foreach (IGrouping<string, XElement> grouping in from n in xml.Element("Form").Descendants()
				where n.Attribute("name") != null && n.Name.LocalName != ComponentType.Item.ToString()
				group n by n.Attribute("name").Value)
			{
				if (this._TSDValidater != null && this._TSDValidater.IsBusy && this._TSDValidater.CancellationPending)
				{
					return;
				}
				if (grouping.Count<XElement>() > 1)
				{
					DocumentErrorsEventArgs e2 = new DocumentErrorsEventArgs();
					e2.ProgramKey = this.Key;
					e2.SourceType = this.Key.PackType;
					e2.ErrorType = ErrorsType.ERROR;
					e2.Time = DateTime.Now;
					e2.Key = grouping.Key;
					e2.Description = string.Format(text, grouping.Key);
					EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e2);
					flag2 = true;
				}
			}
			List<XElement> list = (from sr in SettingManager.Get().GetTzpManger(this.Key).SpecificationInfo.AssociateTable.Source.Descendants("sr")
				where sr.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
				select sr).ToList<XElement>();
			text = Application.Current.FindResource("Message_NotSetTableAssociation") as string;
			string text2 = "s_detail\\\\d{1}$";
			foreach (XElement xelement2 in this._screenRecordManager.GetRecords())
			{
				if (this._TSDValidater != null && this._TSDValidater.IsBusy && this._TSDValidater.CancellationPending)
				{
					return;
				}
				bool flag3 = false;
				string value = xelement2.Attribute("name").Value;
				if (Regex.IsMatch(value, text2))
				{
					foreach (XElement xelement3 in list)
					{
						if (xelement3.Attribute("name").Value == value)
						{
							flag3 = true;
							break;
						}
					}
					if (!flag3)
					{
						DocumentErrorsEventArgs e3 = new DocumentErrorsEventArgs();
						e3.ProgramKey = this.Key;
						e3.SourceType = this.Key.PackType;
						e3.ErrorType = ErrorsType.ERROR;
						e3.Time = DateTime.Now;
						e3.Key = value;
						e3.Description = string.Format(text, value);
						EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Publish(e3);
						flag2 = true;
					}
				}
			}
			this.IsUIValidated = !flag2;
		}

		// Token: 0x06000870 RID: 2160 RVA: 0x00025F3C File Offset: 0x0002413C
		public void SetCitedSTD(string content)
		{
			if (string.IsNullOrEmpty(content))
			{
				return;
			}
			this.CiteSTD = XElement.Parse(content);
		}

		// Token: 0x17000258 RID: 600
		// (get) Token: 0x06000871 RID: 2161 RVA: 0x00025F53 File Offset: 0x00024153
		public IEnumerable<SpecProgRelNode> OtherRrog
		{
			get
			{
				return this.prog_rels;
			}
		}

		// Token: 0x17000259 RID: 601
		// (get) Token: 0x06000872 RID: 2162 RVA: 0x00025F5B File Offset: 0x0002415B
		public IEnumerable<SpecReferenceNode> OtherRef
		{
			get
			{
				return this.ref_fields;
			}
		}

		// Token: 0x1700025A RID: 602
		// (get) Token: 0x06000873 RID: 2163 RVA: 0x00025F63 File Offset: 0x00024163
		public IEnumerable<SpecMultiLangNode> OtherMultiLangs
		{
			get
			{
				return this.multi_langs;
			}
		}

		// Token: 0x1700025B RID: 603
		// (get) Token: 0x06000874 RID: 2164 RVA: 0x00025F6B File Offset: 0x0002416B
		public IEnumerable<SpecHelpCodeNode> OtherHelpCode
		{
			get
			{
				return this.help_codes;
			}
		}

		// Token: 0x1700025C RID: 604
		// (get) Token: 0x06000875 RID: 2165 RVA: 0x00025F73 File Offset: 0x00024173
		public IEnumerable<SpecFieldNode> OtherFields
		{
			get
			{
				return this.fields;
			}
		}

		// Token: 0x1700025D RID: 605
		// (get) Token: 0x06000876 RID: 2166 RVA: 0x00026124 File Offset: 0x00024324
		public IEnumerable<SpecFieldNode> FieldsForView
		{
			get
			{
				foreach (FormSpecModel i in this._formSpecDic.Values)
				{
					if (i.SpecField != null)
					{
						yield return i.SpecField;
					}
				}
				yield break;
			}
		}

		// Token: 0x1700025E RID: 606
		// (get) Token: 0x06000877 RID: 2167 RVA: 0x000262EC File Offset: 0x000244EC
		public IEnumerable<SpecMultiLangNode> MultiLangsForView
		{
			get
			{
				foreach (FormSpecModel i in this._formSpecDic.Values)
				{
					if (i.SpecMultiLang != null)
					{
						yield return i.SpecMultiLang;
					}
				}
				yield break;
			}
		}

		// Token: 0x1700025F RID: 607
		// (get) Token: 0x06000878 RID: 2168 RVA: 0x000264B4 File Offset: 0x000246B4
		public IEnumerable<SpecReferenceNode> ReferenceNodesForView
		{
			get
			{
				foreach (FormSpecModel i in this._formSpecDic.Values)
				{
					if (i.SpecReference != null)
					{
						yield return i.SpecReference;
					}
				}
				yield break;
			}
		}

		// Token: 0x17000260 RID: 608
		// (get) Token: 0x06000879 RID: 2169 RVA: 0x0002667C File Offset: 0x0002487C
		public IEnumerable<SpecProgRelNode> ProgRelNodesForView
		{
			get
			{
				foreach (FormSpecModel i in this._formSpecDic.Values)
				{
					if (i.SpecProgRel != null)
					{
						yield return i.SpecProgRel;
					}
				}
				yield break;
			}
		}

		// Token: 0x17000261 RID: 609
		// (get) Token: 0x0600087A RID: 2170 RVA: 0x0002684C File Offset: 0x00024A4C
		public IEnumerable<SpecActionNode> ActionsForView
		{
			get
			{
				foreach (SpecActionNode act in this._acts)
				{
					if (!act.IsToolBarAction && (act.Status & SpecStatus.DELETE) != SpecStatus.DELETE)
					{
						yield return act;
					}
				}
				yield break;
			}
		}

		// Token: 0x17000262 RID: 610
		// (get) Token: 0x0600087B RID: 2171 RVA: 0x00026A14 File Offset: 0x00024C14
		public IEnumerable<SpecTreeNode> TreesForView
		{
			get
			{
				foreach (FormSpecModel i in this._formSpecDic.Values)
				{
					if (i.SpecTree != null)
					{
						yield return i.SpecTree;
					}
				}
				yield break;
			}
		}

		// Token: 0x17000263 RID: 611
		// (get) Token: 0x0600087C RID: 2172 RVA: 0x00026A31 File Offset: 0x00024C31
		public ReadOnlyObservableCollection<SpecActionNode> Actions
		{
			get
			{
				if (this._readonyActs == null)
				{
					this._readonyActs = new ReadOnlyObservableCollection<SpecActionNode>(this._acts);
				}
				return this._readonyActs;
			}
		}

		// Token: 0x17000264 RID: 612
		// (get) Token: 0x0600087D RID: 2173 RVA: 0x00026A52 File Offset: 0x00024C52
		public IEnumerable<SpecTreeNode> OtherTrees
		{
			get
			{
				return this.trees;
			}
		}

		// Token: 0x17000265 RID: 613
		// (get) Token: 0x0600087E RID: 2174 RVA: 0x00026A5A File Offset: 0x00024C5A
		// (set) Token: 0x0600087F RID: 2175 RVA: 0x00026A62 File Offset: 0x00024C62
		public SpecProgramAll ProgramSpec { get; set; }

		// Token: 0x17000266 RID: 614
		// (get) Token: 0x06000880 RID: 2176 RVA: 0x00026A6B File Offset: 0x00024C6B
		// (set) Token: 0x06000881 RID: 2177 RVA: 0x00026A73 File Offset: 0x00024C73
		public SpecProgramDB ProgramDBSpec { get; set; }

		// Token: 0x17000267 RID: 615
		// (get) Token: 0x06000882 RID: 2178 RVA: 0x00026A7C File Offset: 0x00024C7C
		// (set) Token: 0x06000883 RID: 2179 RVA: 0x00026A84 File Offset: 0x00024C84
		public SpecProgramDI ProgramDISpec { get; set; }

		// Token: 0x17000268 RID: 616
		// (get) Token: 0x06000884 RID: 2180 RVA: 0x00026A8D File Offset: 0x00024C8D
		// (set) Token: 0x06000885 RID: 2181 RVA: 0x00026A95 File Offset: 0x00024C95
		public SpecProgramMI ProgramMISpec { get; set; }

		// Token: 0x06000886 RID: 2182 RVA: 0x00026AA0 File Offset: 0x00024CA0
		private void ConvertTSDToModel()
		{
			XElement xelement = this.TSDElement.Element("toolbar");
			if (xelement != null)
			{
				this.AllowedAction = xelement.Attribute("items").Value.Split(new char[] { ',' });
			}
			this._formSpecDic = new FormSpecDictionary(this.Key);
			this.ConvertFieldsToModel();
			this.ConvertProgRelToModel();
			this.ConvertHelpCodeToModel();
			this.ConvertMultiLangToModel();
			this.ConvertRefToModel();
			this.ConvertActToModel();
			this.ConvertStringsToModel();
			this.ConvertTreesToModel();
			this.ConvertSpecsToMoel();
			this.ConvertExcludeToModel();
			this.ConvertBindingToModel();
			XElement xelement2 = this.FormElement.Element("Form");
			SpecDesignerCommon.ViewModel.XmlElement.BatchAddAttributeCompleter(xelement2);
			this.FormNode = new SpecDesignerCommon.ViewModel.XmlElement(this.Key, xelement2);
			this.GetChildNode(this.FormNode, xelement2);
			this.Add(this.FormNode);
		}

		// Token: 0x06000887 RID: 2183 RVA: 0x00026B90 File Offset: 0x00024D90
		private void TabIndexAutoSort(SpecDesignerCommon.ViewModel.XmlElement parentModel, XElement parent)
		{
			foreach (XElement xelement in parent.Elements())
			{
				SpecDesignerCommon.ViewModel.XmlElement xmlElement = new SpecDesignerCommon.ViewModel.XmlElement(this.Key, xelement);
				for (int i = 0; i < xmlElement.Nodes.Count<SpecDesignerCommon.ViewModel.XmlElement>(); i++)
				{
				}
			}
		}

		// Token: 0x06000888 RID: 2184 RVA: 0x00026BFC File Offset: 0x00024DFC
		private void GetChildNode(SpecDesignerCommon.ViewModel.XmlElement parentModel, XElement parent)
		{
			foreach (XElement xelement in parent.Elements())
			{
				SpecDesignerCommon.ViewModel.XmlElement xmlElement = new SpecDesignerCommon.ViewModel.XmlElement(this.Key, xelement);
				xmlElement.Parent = parentModel;
				this.GetChildNode(xmlElement, xelement);
				ComponentType type = parentModel.Type;
				switch (type)
				{
				case ComponentType.HBox:
					xmlElement.SetInitGridY();
					parentModel.Nodes.Add(xmlElement);
					this.Add(xmlElement);
					continue;
				case ComponentType.Page:
					goto IL_009D;
				case ComponentType.RadioGroup:
					break;
				default:
					if (type == ComponentType.VBox)
					{
						xmlElement.SetInitGridX();
						parentModel.Nodes.Add(xmlElement);
						this.Add(xmlElement);
						continue;
					}
					if (type != ComponentType.ComboBox)
					{
						goto IL_009D;
					}
					break;
				}
				parentModel.Items.Add(xmlElement);
				continue;
				IL_009D:
				if (!(xmlElement.NodeName == "DateTimeEdit") || !(SettingManager.Get().ErpVer == "1.0"))
				{
					parentModel.Nodes.Add(xmlElement);
					this.Add(xmlElement);
				}
			}
		}

		// Token: 0x06000889 RID: 2185 RVA: 0x00026D08 File Offset: 0x00024F08
		private void ConvertTreesToModel()
		{
			foreach (XElement xelement in this.TSDElement.Elements("tree"))
			{
				SpecTreeNode specTreeNode = new SpecTreeNode(this.Key, xelement);
				this.trees.Add(specTreeNode);
			}
		}

		// Token: 0x0600088A RID: 2186 RVA: 0x00026D78 File Offset: 0x00024F78
		private void ConvertActToModel()
		{
			foreach (XElement xelement in this.TSDElement.Elements("act"))
			{
				SpecActionNode specActionNode = new SpecActionNode(this.Key, xelement);
				this._acts.Add(specActionNode);
			}
		}

		// Token: 0x0600088B RID: 2187 RVA: 0x00026DE8 File Offset: 0x00024FE8
		private void ConvertStringsToModel()
		{
			foreach (XElement xelement in this.TSDElement.Elements("strings").Elements<XElement>("sfield"))
			{
				SpecFieldStringNode specFieldStringNode = new SpecFieldStringNode(this.Key, xelement);
				this.fieldStrings.Add(specFieldStringNode);
			}
			foreach (XElement xelement2 in this.TSDElement.Elements("strings").Elements<XElement>("sact"))
			{
				SpecActStringNode specActStringNode = new SpecActStringNode(this.Key, xelement2);
				this.actStrings.Add(specActStringNode);
			}
		}

		// Token: 0x0600088C RID: 2188 RVA: 0x00026EDC File Offset: 0x000250DC
		private void ConvertRefToModel()
		{
			foreach (XElement xelement in this.TSDElement.Elements("ref_field").Elements<XElement>())
			{
				SpecReferenceNode specReferenceNode = new SpecReferenceNode(this.Key, xelement);
				this.ref_fields.Add(specReferenceNode);
			}
		}

		// Token: 0x0600088D RID: 2189 RVA: 0x00026F50 File Offset: 0x00025150
		private void ConvertMultiLangToModel()
		{
			foreach (XElement xelement in this.TSDElement.Elements("multi_lang").Elements<XElement>())
			{
				SpecMultiLangNode specMultiLangNode = new SpecMultiLangNode(this.Key, xelement);
				this.multi_langs.Add(specMultiLangNode);
			}
		}

		// Token: 0x0600088E RID: 2190 RVA: 0x00026FC4 File Offset: 0x000251C4
		private void ConvertHelpCodeToModel()
		{
			foreach (XElement xelement in this.TSDElement.Elements("help_code").Elements<XElement>())
			{
				SpecHelpCodeNode specHelpCodeNode = new SpecHelpCodeNode(this.Key, xelement);
				this.help_codes.Add(specHelpCodeNode);
			}
		}

		// Token: 0x0600088F RID: 2191 RVA: 0x00027038 File Offset: 0x00025238
		private void ConvertProgRelToModel()
		{
			foreach (XElement xelement in this.TSDElement.Elements("prog_rel").Elements<XElement>())
			{
				SpecProgRelNode specProgRelNode = new SpecProgRelNode(this.Key, xelement);
				this.prog_rels.Add(specProgRelNode);
			}
		}

		// Token: 0x06000890 RID: 2192 RVA: 0x000270AC File Offset: 0x000252AC
		private void ConvertFieldsToModel()
		{
			foreach (XElement xelement in this.TSDElement.Elements("field"))
			{
				SpecFieldNode specFieldNode = new SpecFieldNode(this.Key, xelement);
				this.fields.Add(specFieldNode);
			}
		}

		// Token: 0x06000891 RID: 2193 RVA: 0x0002711C File Offset: 0x0002531C
		private void ConvertSpecsToMoel()
		{
			if (this.ProgramSpec != null)
			{
				throw new FileFormatErrorException("Duplicate \"All\" node");
			}
			XElement xelement = this.TSDElement.Descendants("all").FirstOrDefault<XElement>();
			this.ProgramSpec = ((xelement != null) ? new SpecProgramAll(this.Key, xelement) : SpecProgramAll.Create(this));
			xelement = this.TSDElement.Descendants("db_all").FirstOrDefault<XElement>();
			this.ProgramDBSpec = ((xelement != null) ? new SpecProgramDB(this.Key, xelement) : SpecProgramDB.Create(this));
			xelement = this.TSDElement.Descendants("di_all").FirstOrDefault<XElement>();
			this.ProgramDISpec = ((xelement != null) ? new SpecProgramDI(this.Key, xelement) : SpecProgramDI.Create(this));
			xelement = this.TSDElement.Descendants("mi_all").FirstOrDefault<XElement>();
			this.ProgramMISpec = ((xelement != null) ? new SpecProgramMI(this.Key, xelement) : SpecProgramMI.Create(this));
		}

		// Token: 0x06000892 RID: 2194 RVA: 0x0002721C File Offset: 0x0002541C
		private void ConvertExcludeToModel()
		{
			XElement xelement = this.TSDElement.Element("exclude");
			foreach (XElement xelement2 in xelement.Elements("widget"))
			{
				this.excludes.Add(xelement2);
			}
		}

		// Token: 0x06000893 RID: 2195 RVA: 0x000272D0 File Offset: 0x000254D0
		public bool IsActionDefault(string name)
		{
			if (SpecificationInfo.DisabledActions.Contains(name))
			{
				return true;
			}
			if (SettingManager.Get().ActionDefaults == null)
			{
				return false;
			}
			int num = (from ad in SettingManager.Get().ActionDefaults.Elements("ActionDefault")
				where ad.Attribute("name") != null && ad.Attribute("name").Value == name
				select ad).Count<XElement>();
			return num > 0;
		}

		// Token: 0x06000894 RID: 2196 RVA: 0x0002736C File Offset: 0x0002556C
		public bool IsLegalForAction(string name)
		{
			if (this.IsActionDefault(name))
			{
				return false;
			}
			IEnumerable<SpecActionNode> enumerable = this._acts.Where<SpecActionNode>((SpecActionNode a) => a.Name == name && (a.Status & SpecStatus.DELETE) == SpecStatus.NULL);
			return enumerable.Count<SpecActionNode>() == 0;
		}

		// Token: 0x06000895 RID: 2197 RVA: 0x000273B8 File Offset: 0x000255B8
		public void ConvertComponentType(SpecDesignerCommon.ViewModel.XmlElement oldElement, SpecDesignerCommon.ViewModel.XmlElement newElement)
		{
			if (!this.FormSpeDictionary.ContainsKey(oldElement.Name))
			{
				throw new Exception(string.Format("Name：{0} Not exist!", oldElement.Name));
			}
			FormSpecModel formSpecModel = this.FindNodeByName(oldElement.Name);
			SpecDesignerCommon.ViewModel.XmlElement parent = oldElement.Parent;
			int index = oldElement.Index;
			SpecDesignerCommon.ViewModel.XmlElement bindElement = oldElement.BindElement;
			if (bindElement != null)
			{
				bindElement.ClearBinding();
			}
			SpecActionNode specActionNode = ((formSpecModel.SpecAction == null) ? null : SpecActionNode.Create(this.Key, formSpecModel.SpecAction.ToXml(), this.Env));
			SpecFieldNode specFieldNode = ((formSpecModel.SpecField == null) ? null : SpecFieldNode.Create(this.Key, formSpecModel.SpecField.ToXml(), this.Env));
			SpecHelpCodeNode specHelpCodeNode = ((formSpecModel.SpecHelpCode == null) ? null : SpecHelpCodeNode.Create(this.Key, formSpecModel.SpecHelpCode.ToXml(), this.Env));
			SpecMultiLangNode specMultiLangNode = ((formSpecModel.SpecMultiLang == null) ? null : SpecMultiLangNode.Create(this.Key, formSpecModel.SpecMultiLang.ToXml(), this.Env));
			SpecProgRelNode specProgRelNode = ((formSpecModel.SpecProgRel == null) ? null : SpecProgRelNode.Create(this.Key, formSpecModel.SpecProgRel.ToXml(), this.Env));
			SpecReferenceNode specReferenceNode = ((formSpecModel.SpecReference == null) ? null : SpecReferenceNode.Create(this.Key, formSpecModel.SpecReference.ToXml(), this.Env));
			parent.RemoveNode(oldElement);
			this.Remove(oldElement.Name);
			if (specFieldNode != null && oldElement.Type != newElement.Type)
			{
				specFieldNode.Widget = newElement.Type.ToString();
			}
			switch (parent.Type)
			{
			case ComponentType.Table:
			case ComponentType.Tree:
				parent.AddNodeAt(newElement, index);
				break;
			default:
				parent.AddNode(newElement);
				break;
			}
			this.Add(newElement);
			if (bindElement != null)
			{
				bindElement.BindElement = newElement;
				newElement.BindElement = bindElement;
			}
			FormSpecModel formSpecModel2 = this.FindNodeByName(newElement.Name);
			switch (formSpecModel2.SpecNodeType)
			{
			case SpecNodeType.PROGREL:
				if (specProgRelNode != null)
				{
					specProgRelNode.Status = SpecStatus.MODIFY;
					formSpecModel2.SpecProgRel = specProgRelNode;
					SpecNodeTransform.TransformProgRelNode(formSpecModel2.SpecProgRel, newElement);
				}
				break;
			case SpecNodeType.REFERENCE:
				if (specReferenceNode != null)
				{
					specReferenceNode.Status = SpecStatus.MODIFY;
					formSpecModel2.SpecReference = specReferenceNode;
					SpecNodeTransform.TransformReferenceNode(formSpecModel2.SpecReference, newElement);
				}
				break;
			case SpecNodeType.MULTILANG:
				if (specMultiLangNode != null)
				{
					specMultiLangNode.Status = SpecStatus.MODIFY;
					formSpecModel2.SpecMultiLang = specMultiLangNode;
				}
				break;
			}
			if (formSpecModel2.SpecAction != null && specActionNode != null)
			{
				specActionNode.Status = SpecStatus.MODIFY;
				formSpecModel2.SpecAction = specActionNode;
			}
			if (formSpecModel2.SpecField != null && specFieldNode != null)
			{
				specFieldNode.Status = SpecStatus.MODIFY;
				formSpecModel2.SpecField = specFieldNode;
				SpecNodeTransform.TransformTableColumn(formSpecModel2.SpecField, newElement);
				SpecNodeTransform.TransformRequired(formSpecModel2.SpecField, newElement);
			}
			if (formSpecModel2.SpecHelpCode != null && specHelpCodeNode != null)
			{
				specHelpCodeNode.Status = SpecStatus.MODIFY;
				formSpecModel2.SpecHelpCode = specHelpCodeNode;
			}
			newElement.OnPropertyChanged("");
		}

		// Token: 0x06000896 RID: 2198 RVA: 0x000276A0 File Offset: 0x000258A0
		internal void FindAllFieldsSpecByName(FormSpecModel formSpec)
		{
			SpecNodeType specNodeType = formSpec.SpecNodeType;
			string name = formSpec.Name;
			this.FindExcludeNodeByID(formSpec);
			switch (specNodeType)
			{
			case SpecNodeType.FIELD:
			{
				this.FindFieldSpecById(formSpec);
				bool flag = false;
				if (this.GetCodeTemplate() != "Q" && formSpec.GeneroComponent.Parent != null && formSpec.GeneroComponent.Parent.Name != "s_browse")
				{
					switch (formSpec.GeneroComponent.Parent.Type)
					{
					case ComponentType.ScrollGrid:
					case ComponentType.Table:
						this.FindProgRelSpecById(formSpec);
						if (formSpec.SpecField.IsCited)
						{
							formSpec.SpecProgRel.SetIsCited();
						}
						flag = true;
						break;
					}
				}
				if (!flag)
				{
					formSpec.SpecProgRel = null;
				}
				ComponentType type = formSpec.Type;
				if (type != ComponentType.Edit)
				{
					switch (type)
					{
					case ComponentType.Button:
						this.FindActSpecById(formSpec);
						if (formSpec.SpecField.IsCited)
						{
							formSpec.SpecAction.SetIsCited();
							return;
						}
						return;
					case ComponentType.ButtonEdit:
						break;
					default:
						return;
					}
				}
				if (formSpec.GeneroComponent.Parent != null && "s_browse" == formSpec.GeneroComponent.Parent.Name)
				{
					formSpec.SpecHelpCode = null;
					return;
				}
				if (formSpec.GeneroComponent.Parent != null && ComponentType.Tree == formSpec.GeneroComponent.Parent.Type)
				{
					formSpec.SpecHelpCode = null;
					return;
				}
				this.FindHelpCodeSpecById(formSpec);
				if (formSpec.SpecField.IsCited)
				{
					formSpec.SpecHelpCode.SetIsCited();
				}
				break;
			}
			case SpecNodeType.ACTION:
				this.FindActSpecById(formSpec);
				return;
			case SpecNodeType.TOOLBAR:
				this.FindActionSpecIgnoreStatus(formSpec);
				return;
			case SpecNodeType.FORMONLY:
			case SpecNodeType.NONE:
				break;
			case SpecNodeType.PROGREL:
				this.FindProgRelSpecById(formSpec);
				return;
			case SpecNodeType.REFERENCE:
				this.FindReferenceSpecById(formSpec);
				return;
			case SpecNodeType.MULTILANG:
				this.FindMultiLangSpecById(formSpec);
				return;
			case SpecNodeType.TREE:
				this.FindTreeSpecById(formSpec);
				return;
			default:
				return;
			}
		}

		// Token: 0x06000897 RID: 2199 RVA: 0x00027870 File Offset: 0x00025A70
		private void FindTreeSpecById(FormSpecModel formSpec)
		{
			if (formSpec.SpecTree != null)
			{
				return;
			}
			SpecTreeNode specTreeNode = null;
			foreach (SpecTreeNode specTreeNode2 in this.trees)
			{
				if (formSpec.Name.Equals(specTreeNode2.Name) && (specTreeNode2.Status & SpecStatus.DELETE) == SpecStatus.NULL)
				{
					specTreeNode = specTreeNode2;
					this.trees.Remove(specTreeNode2);
					break;
				}
			}
			if (specTreeNode == null)
			{
				specTreeNode = SpecTreeNode.Create(this, formSpec.Name);
			}
			formSpec.SetSpecNode(specTreeNode);
		}

		// Token: 0x06000898 RID: 2200 RVA: 0x0002790C File Offset: 0x00025B0C
		private void FindFieldSpecById(FormSpecModel formSpecModel)
		{
			ComponentType type = formSpecModel.Type;
			if (type <= ComponentType.Canvas)
			{
				switch (type)
				{
				case ComponentType.Folder:
				case ComponentType.Grid:
				case ComponentType.Group:
				case ComponentType.HBox:
				case ComponentType.ScrollGrid:
				case ComponentType.Table:
				case ComponentType.VBox:
				case ComponentType.Label:
					break;
				case ComponentType.Form:
				case ComponentType.Page:
				case ComponentType.RadioGroup:
				case ComponentType.Tree:
					goto IL_0056;
				default:
					if (type != ComponentType.Canvas)
					{
						goto IL_0056;
					}
					break;
				}
			}
			else if (type != ComponentType.Image && type != ComponentType.HLine)
			{
				goto IL_0056;
			}
			return;
			IL_0056:
			if (formSpecModel.SpecField != null)
			{
				return;
			}
			SpecFieldNode specFieldNode = null;
			foreach (SpecFieldNode specFieldNode2 in this.fields)
			{
				if (formSpecModel.Name.Equals(specFieldNode2.Name) && (specFieldNode2.Status & SpecStatus.DELETE) == SpecStatus.NULL)
				{
					specFieldNode = specFieldNode2;
					this.fields.Remove(specFieldNode2);
					break;
				}
			}
			if (specFieldNode == null)
			{
				specFieldNode = SpecFieldNode.Create(this, formSpecModel.Name);
				if (SettingManager.Get().CheckUndoRedoManager(this.Key))
				{
					SpecFieldNode.SetDefaultProperties(this.Key, formSpecModel.Type.ToString(), specFieldNode);
					SpecNodeTransform.TransformRequired(specFieldNode, formSpecModel.GeneroComponent);
				}
			}
			formSpecModel.SetSpecNode(specFieldNode);
		}

		// Token: 0x06000899 RID: 2201 RVA: 0x00027A38 File Offset: 0x00025C38
		private void FindActSpecById(FormSpecModel formSpecModel)
		{
			if (this.IsActionDefault(formSpecModel.Name))
			{
				return;
			}
			if (formSpecModel.SpecAction != null)
			{
				return;
			}
			SpecActionNode specActionNode = null;
			foreach (SpecActionNode specActionNode2 in this._acts)
			{
				if (formSpecModel.Name.Equals(specActionNode2.Name) && (specActionNode2.Status & SpecStatus.DELETE) == SpecStatus.NULL)
				{
					specActionNode = specActionNode2;
					break;
				}
			}
			if (specActionNode == null)
			{
				specActionNode = SpecActionNode.Create(this, formSpecModel.Name);
				this._acts.Add(specActionNode);
			}
			formSpecModel.SetSpecNode(specActionNode);
		}

		// Token: 0x0600089A RID: 2202 RVA: 0x00027AF8 File Offset: 0x00025CF8
		public SpecActionNode FindActionDefaultSpecOrCreate(string name)
		{
			SpecActionNode specActionNode = this._acts.Where<SpecActionNode>((SpecActionNode act) => act.Name == name).FirstOrDefault<SpecActionNode>();
			if (specActionNode == null)
			{
				specActionNode = SpecActionNode.Create(this, name);
				this._acts.Add(specActionNode);
			}
			return specActionNode;
		}

		// Token: 0x0600089B RID: 2203 RVA: 0x00027B4C File Offset: 0x00025D4C
		private void FindActionSpecIgnoreStatus(FormSpecModel formSpecModel)
		{
			if (formSpecModel.SpecAction != null)
			{
				return;
			}
			SpecActionNode specActionNode = null;
			foreach (SpecActionNode specActionNode2 in this._acts)
			{
				if (formSpecModel.Name.Equals(specActionNode2.Name) && (specActionNode2.Status & SpecStatus.DELETE) == SpecStatus.NULL)
				{
					specActionNode = specActionNode2;
					break;
				}
			}
			if (specActionNode == null)
			{
				specActionNode = SpecActionNode.Create(this, formSpecModel.Name);
			}
			formSpecModel.SetSpecNode(specActionNode);
		}

		// Token: 0x0600089C RID: 2204 RVA: 0x00027BD8 File Offset: 0x00025DD8
		private void FindHelpCodeSpecById(FormSpecModel formSpecModel)
		{
			if (formSpecModel.SpecHelpCode != null)
			{
				return;
			}
			SpecHelpCodeNode specHelpCodeNode = null;
			foreach (SpecHelpCodeNode specHelpCodeNode2 in this.help_codes)
			{
				if (formSpecModel.Name.Equals(specHelpCodeNode2.Name) && (specHelpCodeNode2.Status & SpecStatus.DELETE) == SpecStatus.NULL)
				{
					specHelpCodeNode = specHelpCodeNode2;
					this.help_codes.Remove(specHelpCodeNode2);
					break;
				}
			}
			if (specHelpCodeNode == null)
			{
				specHelpCodeNode = SpecHelpCodeNode.Create(this, formSpecModel.Name);
			}
			formSpecModel.SetSpecNode(specHelpCodeNode);
		}

		// Token: 0x0600089D RID: 2205 RVA: 0x00027C74 File Offset: 0x00025E74
		private void FindMultiLangSpecById(FormSpecModel formSpecModel)
		{
			if (formSpecModel.SpecMultiLang != null)
			{
				return;
			}
			SpecMultiLangNode specMultiLangNode = null;
			foreach (SpecMultiLangNode specMultiLangNode2 in this.multi_langs)
			{
				if (formSpecModel.Name.Equals(specMultiLangNode2.Name) && (specMultiLangNode2.Status & SpecStatus.DELETE) == SpecStatus.NULL)
				{
					specMultiLangNode = specMultiLangNode2;
					this.multi_langs.Remove(specMultiLangNode2);
					break;
				}
			}
			if (specMultiLangNode == null)
			{
				specMultiLangNode = SpecMultiLangNode.Create(this, formSpecModel.Name);
			}
			formSpecModel.SetSpecNode(specMultiLangNode);
		}

		// Token: 0x0600089E RID: 2206 RVA: 0x00027D10 File Offset: 0x00025F10
		private void FindProgRelSpecById(FormSpecModel formSpecModel)
		{
			if (formSpecModel.SpecProgRel != null)
			{
				return;
			}
			SpecProgRelNode specProgRelNode = null;
			foreach (SpecProgRelNode specProgRelNode2 in this.prog_rels)
			{
				if (formSpecModel.Name.Equals(specProgRelNode2.Name) && (specProgRelNode2.Status & SpecStatus.DELETE) == SpecStatus.NULL)
				{
					specProgRelNode = specProgRelNode2;
					this.prog_rels.Remove(specProgRelNode2);
					break;
				}
			}
			if (specProgRelNode == null)
			{
				specProgRelNode = SpecProgRelNode.Create(this, formSpecModel.Name);
			}
			formSpecModel.SetSpecNode(specProgRelNode);
		}

		// Token: 0x0600089F RID: 2207 RVA: 0x00027DAC File Offset: 0x00025FAC
		private void FindReferenceSpecById(FormSpecModel formSpecModel)
		{
			if (formSpecModel.SpecReference != null)
			{
				return;
			}
			SpecReferenceNode specReferenceNode = null;
			foreach (SpecReferenceNode specReferenceNode2 in this.ref_fields)
			{
				if (formSpecModel.Name.Equals(specReferenceNode2.Name) && (specReferenceNode2.Status & SpecStatus.DELETE) == SpecStatus.NULL)
				{
					specReferenceNode = specReferenceNode2;
					this.ref_fields.Remove(specReferenceNode2);
					break;
				}
			}
			if (specReferenceNode == null)
			{
				specReferenceNode = SpecReferenceNode.Create(this, formSpecModel.Name);
			}
			formSpecModel.SetSpecNode(specReferenceNode);
		}

		// Token: 0x060008A0 RID: 2208 RVA: 0x00027EB0 File Offset: 0x000260B0
		private void FindExcludeNodeByID(FormSpecModel model)
		{
			if (model.SpecExcludeNode != null)
			{
				return;
			}
			XElement xelement = this.excludes.Where<XElement>((XElement n) => n.Attribute("name").Value == model.Name && !n.Attribute("status").Value.Equals(ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE), StringComparison.CurrentCultureIgnoreCase)).FirstOrDefault<XElement>();
			if (xelement != null)
			{
				this.excludes.Remove(xelement);
				model.SetExcludedNode(xelement);
				return;
			}
			model.SetExcluded(this, false);
		}

		// Token: 0x060008A1 RID: 2209 RVA: 0x00028008 File Offset: 0x00026208
		internal void Remove(AbstractSpecNode node, string oldName)
		{
			if (node is SpecActionNode)
			{
				IEnumerable<SpecActionNode> enumerable = this._acts.Where<SpecActionNode>((SpecActionNode a) => a.Name == oldName);
				while (enumerable != null && enumerable.Count<SpecActionNode>() > 0)
				{
					this._acts.Remove(enumerable.ElementAt<SpecActionNode>(0));
				}
				SpecActionNode specActionNode = SpecActionNode.Create(this, oldName);
				specActionNode.Status = SpecStatus.DELETE;
				this._acts.Insert(0, specActionNode);
			}
			else if (node is SpecFieldNode)
			{
				IEnumerable<SpecFieldNode> enumerable2 = this.fields.Where<SpecFieldNode>((SpecFieldNode a) => a.Name == oldName && (a.Status & SpecStatus.DELETE) == SpecStatus.DELETE);
				while (enumerable2 != null && enumerable2.Count<SpecFieldNode>() > 0)
				{
					this.fields.Remove(enumerable2.ElementAt<SpecFieldNode>(0));
				}
				SpecFieldNode specFieldNode = SpecFieldNode.Create(this, oldName);
				specFieldNode.Status = SpecStatus.DELETE;
				this.fields.Add(specFieldNode);
			}
			else if (node is SpecHelpCodeNode)
			{
				IEnumerable<SpecHelpCodeNode> enumerable3 = this.help_codes.Where<SpecHelpCodeNode>((SpecHelpCodeNode a) => a.Name == oldName && (a.Status & SpecStatus.DELETE) == SpecStatus.DELETE);
				while (enumerable3 != null && enumerable3.Count<SpecHelpCodeNode>() > 0)
				{
					this.help_codes.Remove(enumerable3.ElementAt<SpecHelpCodeNode>(0));
				}
				SpecHelpCodeNode specHelpCodeNode = SpecHelpCodeNode.Create(this, oldName);
				specHelpCodeNode.Status = SpecStatus.DELETE;
				this.help_codes.Add(specHelpCodeNode);
			}
			else if (node is SpecMultiLangNode)
			{
				IEnumerable<SpecMultiLangNode> enumerable4 = this.multi_langs.Where<SpecMultiLangNode>((SpecMultiLangNode a) => a.Name == oldName && (a.Status & SpecStatus.DELETE) == SpecStatus.DELETE);
				while (enumerable4 != null && enumerable4.Count<SpecMultiLangNode>() > 0)
				{
					this.multi_langs.Remove(enumerable4.ElementAt<SpecMultiLangNode>(0));
				}
				SpecMultiLangNode specMultiLangNode = SpecMultiLangNode.Create(this, oldName);
				specMultiLangNode.Status = SpecStatus.DELETE;
				this.multi_langs.Add(specMultiLangNode);
			}
			else if (node is SpecProgRelNode)
			{
				IEnumerable<SpecProgRelNode> enumerable5 = this.prog_rels.Where<SpecProgRelNode>((SpecProgRelNode a) => a.Name == oldName && (a.Status & SpecStatus.DELETE) == SpecStatus.DELETE);
				while (enumerable5 != null && enumerable5.Count<SpecProgRelNode>() > 0)
				{
					this.prog_rels.Remove(enumerable5.ElementAt<SpecProgRelNode>(0));
				}
				SpecProgRelNode specProgRelNode = SpecProgRelNode.Create(this, oldName);
				specProgRelNode.Status = SpecStatus.DELETE;
				this.prog_rels.Add(specProgRelNode);
			}
			else if (node is SpecReferenceNode)
			{
				IEnumerable<SpecReferenceNode> enumerable6 = this.ref_fields.Where<SpecReferenceNode>((SpecReferenceNode a) => a.Name == oldName && (a.Status & SpecStatus.DELETE) == SpecStatus.DELETE);
				while (enumerable6 != null && enumerable6.Count<SpecReferenceNode>() > 0)
				{
					this.ref_fields.Remove(enumerable6.ElementAt<SpecReferenceNode>(0));
				}
				SpecReferenceNode specReferenceNode = SpecReferenceNode.Create(this, oldName);
				specReferenceNode.Status = SpecStatus.DELETE;
				this.ref_fields.Add(specReferenceNode);
			}
			else if (node is SpecTreeNode)
			{
				IEnumerable<SpecTreeNode> enumerable7 = this.trees.Where<SpecTreeNode>((SpecTreeNode a) => a.Name == oldName && (a.Status & SpecStatus.DELETE) == SpecStatus.DELETE);
				while (enumerable7 != null && enumerable7.Count<SpecTreeNode>() > 0)
				{
					this.trees.Remove(enumerable7.ElementAt<SpecTreeNode>(0));
				}
				SpecTreeNode specTreeNode = SpecTreeNode.Create(this, oldName);
				specTreeNode.Status = SpecStatus.DELETE;
				this.trees.Add(specTreeNode);
			}
			if (EventAggregatorManager.ContainsKey(this.Key))
			{
				EventAggregatorManager.Get(this.Key).GetEvent<SpecPropertiesChangedEvent>().Publish(this.Key);
			}
		}

		// Token: 0x060008A2 RID: 2210 RVA: 0x00028390 File Offset: 0x00026590
		private void Remove(SpecExcludeNode specExcludeNode, string oldName)
		{
			if (specExcludeNode == null)
			{
				return;
			}
			SpecExcludeNode specExcludeNode2 = SpecExcludeNode.Create(this, oldName);
			specExcludeNode2.Status = SpecStatus.DELETE;
			this.excludes.Add(specExcludeNode2.ToXml());
		}

		// Token: 0x060008A3 RID: 2211 RVA: 0x000283F0 File Offset: 0x000265F0
		public void Remove(SpecActionNode node)
		{
			node.Status |= SpecStatus.DELETE;
			SpecActStringNode specActStringNode = this.ActionStrings.Where<SpecActStringNode>((SpecActStringNode str) => str.Name == node.Name && (str.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecActStringNode>();
			if (specActStringNode != null)
			{
				specActStringNode.Status |= SpecStatus.DELETE;
			}
			if (EventAggregatorManager.ContainsKey(this.Key))
			{
				EventAggregatorManager.Get(this.Key).GetEvent<SpecPropertiesChangedEvent>().Publish(this.Key);
			}
		}

		// Token: 0x060008A4 RID: 2212 RVA: 0x00028474 File Offset: 0x00026674
		public void Remove(string componentName)
		{
			FormSpecModel formSpecModel = this.FindNodeByName(componentName);
			if (formSpecModel != null)
			{
				this.CloneModelToDeleted(formSpecModel, formSpecModel.Name);
				this.FormSpeDictionary.Remove(componentName);
				this._screenRecordManager.RemoveRecordField(formSpecModel.GeneroComponent);
				switch (formSpecModel.GeneroComponent.Type)
				{
				case ComponentType.ScrollGrid:
				case ComponentType.Table:
				case ComponentType.Tree:
					this.AssociateTable.Remove(componentName);
					break;
				default:
					return;
				}
			}
		}

		// Token: 0x060008A5 RID: 2213 RVA: 0x000284E4 File Offset: 0x000266E4
		public void Add(SpecDesignerCommon.ViewModel.XmlElement xmlElement)
		{
			this.Add(xmlElement, false);
		}

		// Token: 0x060008A6 RID: 2214 RVA: 0x000284F0 File Offset: 0x000266F0
		public void Add(SpecDesignerCommon.ViewModel.XmlElement xmlElement, bool isModifiedStatus)
		{
			FormSpecModel formSpecModel = new FormSpecModel(this.Key, xmlElement);
			if (this.FormSpeDictionary.ContainsKey(formSpecModel.Name))
			{
				throw new Exception("Name is Exist");
			}
			this.FindAllFieldsSpecByName(formSpecModel);
			this.FormSpeDictionary.Add(formSpecModel.Name, formSpecModel);
			SpecNodeType specNodeType = formSpecModel.SpecNodeType;
			if (specNodeType == SpecNodeType.FIELD)
			{
				SpecNodeTransform.TransformFormToSpecField(formSpecModel.SpecField, xmlElement);
				if (isModifiedStatus && !string.IsNullOrEmpty(formSpecModel.SpecField.Table) && !string.IsNullOrEmpty(formSpecModel.SpecField.Column))
				{
					formSpecModel.SpecField.Status |= SpecStatus.MODIFY;
				}
				if (this.DatabaseSource != null)
				{
					this.DatabaseSource.RefreshUsed(this.FormSpeDictionary);
				}
			}
			this._screenRecordManager.AddRecordField(xmlElement);
		}

		// Token: 0x060008A7 RID: 2215 RVA: 0x000285B8 File Offset: 0x000267B8
		public void Add(FormSpecModel formSpecModel)
		{
			if (this.FormSpeDictionary.ContainsKey(formSpecModel.Name))
			{
				throw new Exception("Name is Exist");
			}
			this.FindAllFieldsSpecByName(formSpecModel);
			this.FormSpeDictionary.Add(formSpecModel.Name, formSpecModel);
			formSpecModel.IsCited = false;
			formSpecModel.SpecNodeStatus = SpecStatus.MODIFY;
		}

		// Token: 0x060008A8 RID: 2216 RVA: 0x0002860A File Offset: 0x0002680A
		public void Add(SpecActionNode node)
		{
			this._acts.Add(node);
			if (EventAggregatorManager.ContainsKey(this.Key))
			{
				EventAggregatorManager.Get(this.Key).GetEvent<SpecPropertiesChangedEvent>().Publish(this.Key);
			}
		}

		// Token: 0x060008A9 RID: 2217 RVA: 0x0002868C File Offset: 0x0002688C
		private bool CheckActionNameIsExists(string actionID)
		{
			int num = 0;
			try
			{
				List<string> list = new List<string>();
				string text = SettingManager.Get().LoadToolBar(this.GetClass());
				if (text != null)
				{
					XElement xelement = XElement.Parse(text, LoadOptions.None);
					if (xelement != null)
					{
						IEnumerable<string> enumerable = from tb in xelement.Elements("ToolBarItem").Attributes("name")
							select tb.Value;
						list.AddRange(enumerable.ToList<string>());
					}
				}
				IEnumerable<string> enumerable2 = from a in this.Actions
					where (a.Status & SpecStatus.DELETE) == SpecStatus.NULL
					select a.Name;
				list.AddRange(enumerable2);
				IEnumerable<string> enumerable3 = from a in SettingManager.Get().GetTzpManger(this.Key).ActionDefaults.Elements("ActionDefault")
					select a.Attribute("name").Value;
				list.AddRange(enumerable3);
				num = list.Where<string>((string ac) => ac == actionID).Count<string>();
			}
			catch
			{
			}
			return num > 0;
		}

		// Token: 0x060008AA RID: 2218 RVA: 0x0002889C File Offset: 0x00026A9C
		public void Rename(string oldName, string newName)
		{
			if (oldName == newName)
			{
				return;
			}
			if (!this.FormSpeDictionary.ContainsKey(oldName))
			{
				if (this.CheckActionNameIsExists(newName))
				{
					throw new Exception(Application.Current.FindResource("Message_NameAlreadyExist") as string);
				}
				SpecActionNode specActionNode = this._acts.Where<SpecActionNode>((SpecActionNode a) => a.Name == oldName && (a.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecActionNode>();
				if (specActionNode != null)
				{
					IEnumerable<SpecActionNode> enumerable = this._acts.Where<SpecActionNode>((SpecActionNode a) => a.Name == oldName && (a.Status & SpecStatus.DELETE) == SpecStatus.DELETE);
					while (enumerable != null && enumerable.Count<SpecActionNode>() > 0)
					{
						this._acts.Remove(enumerable.ElementAt<SpecActionNode>(0));
					}
					SpecActionNode specActionNode2 = SpecActionNode.Create(this, oldName);
					specActionNode2.Status = SpecStatus.DELETE;
					this._acts.Insert(0, specActionNode2);
					specActionNode.SetName(null, newName);
					SpecActStringNode specActStringNode = this.actStrings.Where<SpecActStringNode>((SpecActStringNode str) => str.Name == oldName && (str.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecActStringNode>();
					if (specActStringNode != null)
					{
						IEnumerable<SpecActStringNode> enumerable2 = this.actStrings.Where<SpecActStringNode>((SpecActStringNode a) => a.Name == oldName && (a.Status & SpecStatus.DELETE) == SpecStatus.DELETE);
						while (enumerable2 != null && enumerable2.Count<SpecActStringNode>() > 0)
						{
							this.actStrings.Remove(enumerable2.ElementAt<SpecActStringNode>(0));
						}
						SpecActStringNode specActStringNode2 = SpecActStringNode.Create(this, oldName);
						specActStringNode2.Status = SpecStatus.DELETE;
						this.actStrings.Insert(0, specActStringNode2);
						specActStringNode.SetName(newName);
						specActStringNode.Status |= SpecStatus.MODIFY;
					}
					if (EventAggregatorManager.ContainsKey(this.Key))
					{
						EventAggregatorManager.Get(this.Key).GetEvent<SpecPropertiesChangedEvent>().Publish(this.Key);
					}
				}
				return;
			}
			else
			{
				FormSpecModel formSpecModel = this.FormSpeDictionary[oldName];
				if (string.IsNullOrEmpty(newName))
				{
					throw new Exception("'name' field is required");
				}
				if (this.IsExists(newName))
				{
					throw new Exception(Application.Current.FindResource("Message_NameAlreadyExist") as string);
				}
				if (Regex.IsMatch(newName, "(?![a-zA-Z0-9_\\.]+)."))
				{
					throw new Exception(Application.Current.FindResource("Message_FieldNameSyntaxIncorrent") as string);
				}
				this._screenRecordManager.RemoveRecordField(formSpecModel.GeneroComponent);
				formSpecModel.Name = newName;
				if (formSpecModel.Name == newName)
				{
					if (!this.FormSpeDictionary.ContainsKey(oldName))
					{
						return;
					}
					this.FormSpeDictionary.Remove(oldName);
					this.CloneModelToDeleted(formSpecModel, oldName);
					this.FormSpeDictionary.Add(newName, formSpecModel);
				}
				SpecNodeTransform.TransformFieldType(formSpecModel.GeneroComponent);
				this._screenRecordManager.AddRecordField(formSpecModel.GeneroComponent);
				if (EventAggregatorManager.ContainsKey(this.Key))
				{
					EventAggregatorManager.Get(this.Key).GetEvent<SpecPropertiesChangedEvent>().Publish(this.Key);
				}
				return;
			}
		}

		// Token: 0x060008AB RID: 2219 RVA: 0x00028BA0 File Offset: 0x00026DA0
		public FormSpecModel FindNodeByName(string name)
		{
			if (!string.IsNullOrEmpty(name) && this.FormSpeDictionary.ContainsKey(name))
			{
				return this.FormSpeDictionary[name];
			}
			return null;
		}

		// Token: 0x060008AC RID: 2220 RVA: 0x00028BC8 File Offset: 0x00026DC8
		public SpecNodeType GetSpecType(SpecDesignerCommon.ViewModel.XmlElement form)
		{
			FormSpecModel formSpecModel = this.FindNodeByName(form.Name);
			if (formSpecModel != null)
			{
				return formSpecModel.SpecNodeType;
			}
			return SpecNodeType.NONE;
		}

		// Token: 0x17000269 RID: 617
		// (get) Token: 0x060008AD RID: 2221 RVA: 0x00028DA0 File Offset: 0x00026FA0
		public IEnumerable<string> ExcludeNodes
		{
			get
			{
				foreach (KeyValuePair<string, FormSpecModel> pair in this._formSpecDic)
				{
					KeyValuePair<string, FormSpecModel> keyValuePair = pair;
					if (keyValuePair.Value.IsExcluded)
					{
						KeyValuePair<string, FormSpecModel> keyValuePair2 = pair;
						yield return keyValuePair2.Key;
					}
				}
				yield break;
			}
		}

		// Token: 0x060008AE RID: 2222 RVA: 0x00028DC0 File Offset: 0x00026FC0
		internal void CloneModelToDeleted(FormSpecModel model, string oldName)
		{
			if (model == null)
			{
				return;
			}
			if (model.SpecAction != null)
			{
				this.Remove(model.SpecAction, oldName);
			}
			if (model.SpecField != null)
			{
				this.Remove(model.SpecField, oldName);
			}
			if (model.SpecHelpCode != null)
			{
				this.Remove(model.SpecHelpCode, oldName);
			}
			if (model.SpecMultiLang != null)
			{
				this.Remove(model.SpecMultiLang, oldName);
			}
			if (model.SpecProgRel != null)
			{
				this.Remove(model.SpecProgRel, oldName);
			}
			if (model.SpecReference != null)
			{
				this.Remove(model.SpecReference, oldName);
			}
			if (model.SpecTree != null)
			{
				this.Remove(model.SpecTree, oldName);
			}
			if (model.SpecExcludeNode != null)
			{
				this.Remove(model.SpecExcludeNode, oldName);
			}
		}

		// Token: 0x060008AF RID: 2223 RVA: 0x00028E7C File Offset: 0x0002707C
		public bool IsExists(string componentNme)
		{
			bool flag = false;
			if (this.FormSpeDictionary.ContainsKey(componentNme))
			{
				flag = true;
			}
			return flag;
		}

		// Token: 0x060008B0 RID: 2224 RVA: 0x00028E9C File Offset: 0x0002709C
		private bool ContainsSetting(FormSpecModel model, string name)
		{
			return model.Contains("DF", "bcme_t.bcme001");
		}

		// Token: 0x060008B1 RID: 2225 RVA: 0x00028ECC File Offset: 0x000270CC
		public XElement SaveToTSD()
		{
			XElement xelement = new XElement(this.TSDElement);
			if (xelement.Elements("tree") != null)
			{
				xelement.Elements("tree").Remove<XElement>();
			}
			if (xelement.Elements("field") != null)
			{
				xelement.Elements("field").Remove<XElement>();
			}
			if (xelement.Elements("act") != null)
			{
				xelement.Elements("act").Remove<XElement>();
			}
			if (xelement.Element("multi_lang") != null)
			{
				xelement.Element("multi_lang").Elements().Remove<XElement>();
			}
			if (xelement.Element("help_code") != null)
			{
				xelement.Element("help_code").Elements().Remove<XElement>();
			}
			if (xelement.Element("ref_field") != null)
			{
				xelement.Element("ref_field").Elements().Remove<XElement>();
			}
			if (xelement.Element("prog_rel") != null)
			{
				xelement.Element("prog_rel").Elements().Remove<XElement>();
			}
			if (xelement.Element("strings") != null)
			{
				xelement.Element("strings").Elements().Remove<XElement>();
			}
			if (xelement.Element("table") != null)
			{
				xelement.Element("table").Remove();
			}
			if (xelement.Element("all") != null)
			{
				xelement.Element("all").Remove();
			}
			if (xelement.Element("mi_all") != null)
			{
				xelement.Element("mi_all").Remove();
			}
			if (xelement.Element("db_all") != null)
			{
				xelement.Element("db_all").Remove();
			}
			if (xelement.Element("di_all") != null)
			{
				xelement.Element("di_all").Remove();
			}
			XElement xelement2 = xelement.Element("exclude");
			xelement2.RemoveNodes();
			if (xelement.Element("exclude") != null)
			{
				xelement.Element("exclude").Remove();
			}
			XElement xelement3 = new XElement(xelement);
			if (xelement3.Element("toolbar") != null)
			{
				xelement3.Element("toolbar").Remove();
			}
			if (xelement3.Element("sa_spec") != null)
			{
				xelement3.Element("sa_spec").Remove();
			}
			if (xelement3.Element("other") != null)
			{
				xelement3.Element("other").Remove();
			}
			XElement xelement4 = new XElement(this.AssociateTable.Source);
			int i = 0;
			while (i < xelement4.Elements().Count<XElement>())
			{
				XElement xelement5 = xelement4.Elements().ElementAt<XElement>(i);
				List<XElement> list = (from n in xelement5.Elements()
					where string.IsNullOrEmpty(n.Attribute("status").Value)
					select n).ToList<XElement>();
				foreach (XElement xelement6 in list)
				{
					xelement6.Remove();
				}
				if (xelement5.Elements().Count<XElement>() == 0 && string.IsNullOrEmpty(xelement5.Attribute("status").Value))
				{
					xelement5.Remove();
				}
				else
				{
					i++;
				}
			}
			xelement3.Add(xelement4);
			if ((this.ProgramSpec.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
			{
				xelement3.Add(this.ProgramSpec.ToXml());
			}
			if ((this.ProgramMISpec.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
			{
				xelement3.Add(this.ProgramMISpec.ToXml());
			}
			if ((this.ProgramDISpec.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
			{
				xelement3.Add(this.ProgramDISpec.ToXml());
			}
			if ((this.ProgramDBSpec.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
			{
				xelement3.Add(this.ProgramDBSpec.ToXml());
			}
			xelement.Add(new XElement(this.AssociateTable.Source));
			xelement.Add(this.ProgramSpec.ToXml());
			xelement.Add(this.ProgramMISpec.ToXml());
			xelement.Add(this.ProgramDISpec.ToXml());
			xelement.Add(this.ProgramDBSpec.ToXml());
			XElement xelement7 = xelement.Element("strings");
			foreach (SpecActStringNode specActStringNode in this.actStrings)
			{
				xelement7.Add(specActStringNode.ToXml());
				if ((specActStringNode.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
				{
					xelement3.Element("strings").Add(specActStringNode.ToXml());
				}
			}
			foreach (SpecFieldStringNode specFieldStringNode in this.fieldStrings)
			{
				xelement7.Add(specFieldStringNode.ToXml());
				if ((specFieldStringNode.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
				{
					xelement3.Element("strings").Add(specFieldStringNode.ToXml());
				}
			}
			foreach (SpecFieldNode specFieldNode in this.fields)
			{
				xelement.Add(specFieldNode.ToXml());
				if ((specFieldNode.Status & SpecStatus.MODIFY) == SpecStatus.DELETE)
				{
					xelement3.Add(specFieldNode.ToXml());
				}
			}
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair in this._formSpecDic)
			{
				if (keyValuePair.Value.SpecField != null)
				{
					xelement.Add(keyValuePair.Value.SpecField.ToXml());
					if ((keyValuePair.Value.SpecField.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
					{
						xelement3.Add(keyValuePair.Value.SpecField.ToXml());
					}
				}
			}
			foreach (SpecActionNode specActionNode in this._acts)
			{
				if ((specActionNode.Status & SpecStatus.DELETE) == SpecStatus.DELETE)
				{
					xelement.Add(specActionNode.ToXml());
					xelement3.Add(specActionNode.ToXml());
				}
			}
			foreach (SpecActionNode specActionNode2 in this._acts)
			{
				if ((specActionNode2.Status & SpecStatus.DELETE) != SpecStatus.DELETE)
				{
					xelement.Add(specActionNode2.ToXml());
					if ((specActionNode2.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
					{
						xelement3.Add(specActionNode2.ToXml());
					}
				}
			}
			XElement xelement8 = xelement.Element("help_code");
			foreach (SpecHelpCodeNode specHelpCodeNode in this.help_codes)
			{
				xelement8.Add(specHelpCodeNode.ToXml());
				if ((specHelpCodeNode.Status & SpecStatus.DELETE) == SpecStatus.DELETE)
				{
					xelement3.Element("help_code").Add(specHelpCodeNode.ToXml());
				}
			}
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair2 in this._formSpecDic)
			{
				if (keyValuePair2.Value.SpecHelpCode != null)
				{
					xelement8.Add(keyValuePair2.Value.SpecHelpCode.ToXml());
					if ((keyValuePair2.Value.SpecHelpCode.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
					{
						xelement3.Element("help_code").Add(keyValuePair2.Value.SpecHelpCode.ToXml());
					}
				}
			}
			XElement xelement9 = xelement.Element("multi_lang");
			foreach (SpecMultiLangNode specMultiLangNode in this.multi_langs)
			{
				xelement9.Add(specMultiLangNode.ToXml());
				xelement3.Element("multi_lang").Add(specMultiLangNode.ToXml());
			}
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair3 in this._formSpecDic)
			{
				if (keyValuePair3.Value.SpecMultiLang != null)
				{
					xelement9.Add(keyValuePair3.Value.SpecMultiLang.ToXml());
					if ((keyValuePair3.Value.SpecMultiLang.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
					{
						xelement3.Element("multi_lang").Add(keyValuePair3.Value.SpecMultiLang.ToXml());
					}
				}
			}
			XElement xelement10 = xelement.Element("prog_rel");
			foreach (SpecProgRelNode specProgRelNode in this.prog_rels)
			{
				xelement10.Add(specProgRelNode.ToXml());
				xelement3.Element("prog_rel").Add(specProgRelNode.ToXml());
			}
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair4 in this._formSpecDic)
			{
				if (keyValuePair4.Value.SpecProgRel != null)
				{
					xelement10.Add(keyValuePair4.Value.SpecProgRel.ToXml());
					if ((keyValuePair4.Value.SpecProgRel.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
					{
						xelement3.Element("prog_rel").Add(keyValuePair4.Value.SpecProgRel.ToXml());
					}
				}
			}
			XElement xelement11 = xelement.Element("ref_field");
			foreach (SpecReferenceNode specReferenceNode in this.ref_fields)
			{
				xelement11.Add(specReferenceNode.ToXml());
				xelement3.Element("ref_field").Add(specReferenceNode.ToXml());
			}
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair5 in this._formSpecDic)
			{
				if (keyValuePair5.Value.SpecReference != null)
				{
					xelement11.Add(keyValuePair5.Value.SpecReference.ToXml());
					if ((keyValuePair5.Value.SpecReference.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
					{
						xelement3.Element("ref_field").Add(keyValuePair5.Value.SpecReference.ToXml());
					}
				}
			}
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair6 in this._formSpecDic)
			{
				if (keyValuePair6.Value.SpecTree != null)
				{
					xelement.Add(keyValuePair6.Value.SpecTree.ToXml());
					if ((keyValuePair6.Value.SpecTree.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY)
					{
						xelement3.Add(keyValuePair6.Value.SpecTree.ToXml());
					}
				}
			}
			if (xelement2 == null)
			{
				xelement2 = new XElement("exclude", new object[]
				{
					new XAttribute("ver", ""),
					new XAttribute("cite_std", ""),
					new XAttribute("src", "")
				});
				xelement.Add(xelement2);
			}
			xelement3.Add(XElement.Parse(xelement2.ToString()));
			foreach (XElement xelement12 in this.excludes)
			{
				xelement2.Add(xelement12);
			}
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair7 in this.FormSpeDictionary)
			{
				FormSpecModel value = keyValuePair7.Value;
				if (value.SpecExcludeNode != null)
				{
					xelement2.Add(value.SpecExcludeNode.ToXml());
					if ((value.SpecExcludeNode.Status & SpecStatus.MODIFY) == SpecStatus.MODIFY || (value.SpecExcludeNode.Status & SpecStatus.DELETE) == SpecStatus.DELETE)
					{
						xelement3.Element("exclude").Add(value.SpecExcludeNode.ToXml());
					}
				}
			}
			xelement.Add(xelement2);
			this.TSDElement = xelement;
			this.TSD2Element = xelement3;
			return this.TSDElement;
		}

		// Token: 0x060008B2 RID: 2226 RVA: 0x00029CB4 File Offset: 0x00027EB4
		public XElement SaveToForm()
		{
			XElement formElement = this.FormElement;
			XElement xelement = new XElement(formElement);
			xelement.Element("Form").Remove();
			if (xelement.Element("DiagramLayout") != null)
			{
				xelement.Element("DiagramLayout").Remove();
			}
			this.RebuildScreenRecord(xelement);
			xelement.Add(this.FormNode.ToXML());
			this.FormElement = xelement;
			return this.FormElement;
		}

		// Token: 0x060008B3 RID: 2227 RVA: 0x00029D30 File Offset: 0x00027F30
		private void RebuildScreenRecord(XElement form)
		{
			form.Elements("Record").Remove<XElement>();
			if (this._screenRecordManager == null)
			{
				this._screenRecordManager = new ScreenRecordManager(null);
			}
			this._screenRecordManager.Clear();
			this.GetScreenRecord(this.FormNode);
			foreach (XElement xelement in this._screenRecordManager.GetRecords())
			{
				form.Add(xelement);
			}
		}

		// Token: 0x060008B4 RID: 2228 RVA: 0x00029DC4 File Offset: 0x00027FC4
		private void GetScreenRecord(SpecDesignerCommon.ViewModel.XmlElement node)
		{
			foreach (SpecDesignerCommon.ViewModel.XmlElement xmlElement in node.Nodes)
			{
				if (xmlElement.GetAttribute("fieldId") != null)
				{
					List<XElement> list = this._screenRecordManager.GetRecords().ToList<XElement>();
					xmlElement.SetAttribute("fieldId", ScreenRecordManager.GetNewFieldIdRef(list).ToString());
				}
				this._screenRecordManager.AddRecordField(xmlElement);
				this.GetScreenRecord(xmlElement);
			}
		}

		// Token: 0x1700026A RID: 618
		// (get) Token: 0x060008B5 RID: 2229 RVA: 0x00029E54 File Offset: 0x00028054
		public IEnumerable<SpecActStringNode> ActionStrings
		{
			get
			{
				return this.actStrings;
			}
		}

		// Token: 0x060008B6 RID: 2230 RVA: 0x00029EA8 File Offset: 0x000280A8
		public string GetActLocalStringText(string name)
		{
			SpecActStringNode specActStringNode = this.actStrings.Where<SpecActStringNode>((SpecActStringNode str) => str.Name == name && (str.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecActStringNode>();
			if (specActStringNode != null)
			{
				return specActStringNode.Text;
			}
			XElement xelement = (from ad in SettingManager.Get().GetTzpManger(this.Key).ActionDefaults.Descendants("ActionDefault")
				where name.Equals(ad.Attribute("name").Value)
				select ad).FirstOrDefault<XElement>();
			if (xelement != null)
			{
				return xelement.Attribute("text").Value;
			}
			return null;
		}

		// Token: 0x060008B7 RID: 2231 RVA: 0x00029F68 File Offset: 0x00028168
		public void SetActLocalStringText(string fieldName, string text)
		{
			SpecActStringNode specActStringNode = this.actStrings.Where<SpecActStringNode>((SpecActStringNode str) => str.Name == fieldName && (str.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecActStringNode>();
			if (specActStringNode == null)
			{
				specActStringNode = SpecActStringNode.Create(this, fieldName);
				this.actStrings.Add(specActStringNode);
			}
			specActStringNode.Text = text;
			if (EventAggregatorManager.ContainsKey(this.Key))
			{
				EventAggregatorManager.Get(this.Key).GetEvent<SpecPropertiesChangedEvent>().Publish(this.Key);
			}
		}

		// Token: 0x060008B8 RID: 2232 RVA: 0x0002A038 File Offset: 0x00028238
		public void DeleteActLocalStringText(string name)
		{
			SpecActStringNode specActStringNode = this.actStrings.Where<SpecActStringNode>((SpecActStringNode str) => str.Name == name && (str.Status & SpecStatus.DELETE) == SpecStatus.DELETE).FirstOrDefault<SpecActStringNode>();
			if (specActStringNode != null)
			{
				return;
			}
			SpecActStringNode specActStringNode2 = this.actStrings.Where<SpecActStringNode>((SpecActStringNode str) => str.Name == name && (str.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecActStringNode>();
			if (specActStringNode2 != null)
			{
				specActStringNode2.Status |= SpecStatus.DELETE;
			}
			if (EventAggregatorManager.ContainsKey(this.Key))
			{
				EventAggregatorManager.Get(this.Key).GetEvent<SpecPropertiesChangedEvent>().Publish(this.Key);
			}
		}

		// Token: 0x1700026B RID: 619
		// (get) Token: 0x060008B9 RID: 2233 RVA: 0x0002A0C9 File Offset: 0x000282C9
		public IEnumerable<SpecFieldStringNode> FieldStrings
		{
			get
			{
				return this.fieldStrings;
			}
		}

		// Token: 0x060008BA RID: 2234 RVA: 0x0002A0FC File Offset: 0x000282FC
		public string GetFieldLocalStringText(string name)
		{
			SpecFieldStringNode specFieldStringNode = this.fieldStrings.Where<SpecFieldStringNode>((SpecFieldStringNode str) => str.Name == name && (str.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecFieldStringNode>();
			if (specFieldStringNode == null)
			{
				return null;
			}
			return specFieldStringNode.Text;
		}

		// Token: 0x060008BB RID: 2235 RVA: 0x0002A168 File Offset: 0x00028368
		public string GetItemLocalStringText(string name)
		{
			SpecFieldStringNode specFieldStringNode = this.fieldStrings.Where<SpecFieldStringNode>((SpecFieldStringNode str) => str.Name == name && (str.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecFieldStringNode>();
			if (specFieldStringNode == null)
			{
				return null;
			}
			return specFieldStringNode.Name;
		}

		// Token: 0x060008BC RID: 2236 RVA: 0x0002A1D4 File Offset: 0x000283D4
		public void SetFieldLocalStringText(string fieldName, string text)
		{
			if (string.IsNullOrEmpty(fieldName))
			{
				return;
			}
			SpecFieldStringNode specFieldStringNode = this.fieldStrings.Where<SpecFieldStringNode>((SpecFieldStringNode str) => str.Name == fieldName && (str.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecFieldStringNode>();
			if (specFieldStringNode == null)
			{
				specFieldStringNode = SpecFieldStringNode.Create(this, fieldName);
				this.fieldStrings.Add(specFieldStringNode);
			}
			specFieldStringNode.Text = text;
			if (EventAggregatorManager.ContainsKey(this.Key))
			{
				EventAggregatorManager.Get(this.Key).GetEvent<SpecPropertiesChangedEvent>().Publish(this.Key);
			}
		}

		// Token: 0x060008BD RID: 2237 RVA: 0x0002A2B0 File Offset: 0x000284B0
		public void DeleteFieldLocalStringText(string name)
		{
			SpecFieldStringNode specFieldStringNode = this.fieldStrings.Where<SpecFieldStringNode>((SpecFieldStringNode str) => str.Name == name && (str.Status & SpecStatus.DELETE) == SpecStatus.DELETE).FirstOrDefault<SpecFieldStringNode>();
			if (specFieldStringNode != null)
			{
				return;
			}
			SpecFieldStringNode specFieldStringNode2 = this.fieldStrings.Where<SpecFieldStringNode>((SpecFieldStringNode str) => str.Name == name && (str.Status & SpecStatus.DELETE) == SpecStatus.NULL).FirstOrDefault<SpecFieldStringNode>();
			if (specFieldStringNode2 != null)
			{
				specFieldStringNode2.Status |= SpecStatus.DELETE;
			}
			if (EventAggregatorManager.ContainsKey(this.Key))
			{
				EventAggregatorManager.Get(this.Key).GetEvent<SpecPropertiesChangedEvent>().Publish(this.Key);
			}
		}

		// Token: 0x060008BE RID: 2238 RVA: 0x0002A341 File Offset: 0x00028541
		public IEnumerable<XElement> GetRecords()
		{
			this.SaveToForm();
			if (this._screenRecordManager != null)
			{
				return this._screenRecordManager.GetRecords();
			}
			return null;
		}

		// Token: 0x060008BF RID: 2239 RVA: 0x0002A360 File Offset: 0x00028560
		public XElement FindRecord(string ComponentName)
		{
			XElement xelement = null;
			foreach (XElement xelement2 in this.GetRecords())
			{
				xelement = xelement2;
				string text = "";
				foreach (XElement xelement3 in xelement2.Elements())
				{
					text = xelement3.Attribute("name").Value;
					if (text == ComponentName)
					{
						break;
					}
				}
				if (text == ComponentName)
				{
					break;
				}
			}
			return xelement;
		}

		// Token: 0x060008C0 RID: 2240 RVA: 0x0002A418 File Offset: 0x00028618
		public SpecDesignerCommon.ViewModel.XmlElement FindContainer(SpecDesignerCommon.ViewModel.XmlElement Component)
		{
			SpecDesignerCommon.ViewModel.XmlElement xmlElement = Component;
			SpecDesignerCommon.ViewModel.XmlElement xmlElement2 = null;
			while (xmlElement.Parent != null)
			{
				xmlElement = xmlElement.Parent;
				if (xmlElement.Type == ComponentType.Folder || xmlElement.Type == ComponentType.Form || xmlElement.Type == ComponentType.Grid || xmlElement.Type == ComponentType.Group || xmlElement.Type == ComponentType.HBox || xmlElement.Type == ComponentType.Page || xmlElement.Type == ComponentType.VBox)
				{
					xmlElement2 = xmlElement;
					break;
				}
			}
			return xmlElement2;
		}

		// Token: 0x060008C1 RID: 2241 RVA: 0x0002A4A0 File Offset: 0x000286A0
		public IEnumerable<string> GetDependFieldsSource(SpecDesignerCommon.ViewModel.XmlElement formElement)
		{
			List<string> list = new List<string>();
			if (formElement.Parent == null)
			{
				return list;
			}
			switch (formElement.Parent.Type)
			{
			case ComponentType.ScrollGrid:
			case ComponentType.Table:
			case ComponentType.Tree:
			{
				IEnumerable<string> enumerable = from c in formElement.Parent.Nodes
					where !c.HasCantDelTags(c.GetAttribute("tag"))
					select c.Name;
				list.AddRange(enumerable);
				break;
			}
			default:
			{
				IEnumerable<string> undefinedField = this.GetUndefinedField(this.FormNode);
				list.AddRange(undefinedField);
				break;
			}
			}
			list.Remove(formElement.Name);
			return list;
		}

		// Token: 0x060008C2 RID: 2242 RVA: 0x0002A97C File Offset: 0x00028B7C
		private IEnumerable<string> GetUndefinedField(SpecDesignerCommon.ViewModel.XmlElement element)
		{
			ComponentType type = element.Type;
			if (type != ComponentType.Form)
			{
				switch (type)
				{
				case ComponentType.ScrollGrid:
				case ComponentType.Table:
				case ComponentType.Tree:
					break;
				default:
					if (FormDesignSetting.IsContainer(element.NodeName))
					{
						foreach (SpecDesignerCommon.ViewModel.XmlElement c in element.Nodes)
						{
							foreach (string s in this.GetUndefinedField(c))
							{
								yield return s;
							}
						}
					}
					else if (element.GetAttribute("fieldId") != null && !element.IsCantDel)
					{
						yield return element.Name;
					}
					break;
				}
			}
			else
			{
				foreach (SpecDesignerCommon.ViewModel.XmlElement c2 in element.Nodes)
				{
					foreach (string s2 in this.GetUndefinedField(c2))
					{
						yield return s2;
					}
				}
			}
			yield break;
		}

		// Token: 0x060008C3 RID: 2243 RVA: 0x0002A9A0 File Offset: 0x00028BA0
		public IEnumerable<XElement> GetScreenRecords()
		{
			return this._screenRecordManager.GetRecords();
		}

		// Token: 0x060008C4 RID: 2244 RVA: 0x0002A9B0 File Offset: 0x00028BB0
		public void SetFreeStyle()
		{
			XElement xelement = this.TSDElement.Element("other");
			if (xelement == null)
			{
				return;
			}
			xelement.Element("free_style").SetAttributeValue("value", "Y");
			xelement.Element("free_style").SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY));
		}

		// Token: 0x060008C5 RID: 2245 RVA: 0x0002AA25 File Offset: 0x00028C25
		public string GetClass()
		{
			if (this.TSDElement.Attribute("class") != null)
			{
				return this.TSDElement.Attribute("class").Value;
			}
			return "";
		}

		// Token: 0x060008C6 RID: 2246 RVA: 0x0002AA60 File Offset: 0x00028C60
		internal string GetNewActionID()
		{
			string text = "action_";
			List<int> list = new List<int>();
			string text2 = "(?<seq>\\d+$)";
			foreach (SpecActionNode specActionNode in this._acts)
			{
				if (Regex.IsMatch(specActionNode.Name, text2) && (specActionNode.Status & SpecStatus.DELETE) != SpecStatus.DELETE)
				{
					list.Add((int)short.Parse(Regex.Matches(specActionNode.Name, text2)[0].Groups["seq"].Value));
				}
			}
			if (list.Count == 0)
			{
				text += "1";
			}
			else
			{
				list.Sort();
				int num = 0;
				foreach (int num2 in list)
				{
					if (num == 0)
					{
						num = num2;
					}
					if (num2 - num > 1)
					{
						break;
					}
					num = num2;
				}
				text += (num + 1).ToString();
			}
			return text;
		}

		// Token: 0x060008C7 RID: 2247 RVA: 0x0002AB88 File Offset: 0x00028D88
		public string GetCodeTemplate()
		{
			XElement xelement = this.TSDElement.Element("other");
			if (xelement == null)
			{
				return null;
			}
			return xelement.Element("code_template").Attribute("value").Value;
		}

		// Token: 0x060008C8 RID: 2248 RVA: 0x0002ABD4 File Offset: 0x00028DD4
		public void SetCodeTemplate(string codeTemplate)
		{
			XElement xelement = this.TSDElement.Element("other");
			if (xelement == null)
			{
				return;
			}
			xelement.Element("code_template").SetAttributeValue("value", codeTemplate);
			xelement.Element("code_template").SetAttributeValue("status", ReflectionHelpers.GetCustomDescription(SpecStatus.MODIFY));
		}

		// Token: 0x14000020 RID: 32
		// (add) Token: 0x060008C9 RID: 2249 RVA: 0x0002AC48 File Offset: 0x00028E48
		// (remove) Token: 0x060008CA RID: 2250 RVA: 0x0002AC80 File Offset: 0x00028E80
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060008CB RID: 2251 RVA: 0x0002ACB5 File Offset: 0x00028EB5
		private void OnPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x1700026C RID: 620
		// (get) Token: 0x060008CC RID: 2252 RVA: 0x0002ACD4 File Offset: 0x00028ED4
		public bool IsFreeStyle
		{
			get
			{
				return string.Equals(this.TSDElement.Element("other").Element("free_style").Attribute("value")
					.Value, "Y", StringComparison.CurrentCultureIgnoreCase);
			}
		}

		// Token: 0x1700026D RID: 621
		// (get) Token: 0x060008CD RID: 2253 RVA: 0x0002AD29 File Offset: 0x00028F29
		// (set) Token: 0x060008CE RID: 2254 RVA: 0x0002AD31 File Offset: 0x00028F31
		public List<TBinding> SpecBinding { get; set; }

		// Token: 0x060008CF RID: 2255 RVA: 0x0002AD3C File Offset: 0x00028F3C
		private void ConvertBindingToModel()
		{
			if (this._tzpManager.ElementBindings != null)
			{
				if (this.SpecBinding == null)
				{
					this.SpecBinding = new List<TBinding>();
				}
				this.SpecBinding.Clear();
				XmlSerializer xmlSerializer = new XmlSerializer(typeof(SpecBindingInfo));
				xmlSerializer.UnknownNode += this.Serializer_UnknownNode;
				xmlSerializer.UnknownAttribute += this.Serializer_UnknownAttribute;
				SpecBindingInfo specBindingInfo;
				using (TextReader textReader = new StringReader(this._tzpManager.ElementBindings))
				{
					specBindingInfo = (SpecBindingInfo)xmlSerializer.Deserialize(textReader);
				}
				foreach (TBinding tbinding in specBindingInfo.ObjectBindings.bindings)
				{
					this.SpecBinding.Add(tbinding);
				}
			}
		}

		// Token: 0x060008D0 RID: 2256 RVA: 0x0002AE18 File Offset: 0x00029018
		private void Serializer_UnknownAttribute(object sender, XmlAttributeEventArgs e)
		{
			throw new NotImplementedException();
		}

		// Token: 0x060008D1 RID: 2257 RVA: 0x0002AE1F File Offset: 0x0002901F
		private void Serializer_UnknownNode(object sender, XmlNodeEventArgs e)
		{
			throw new NotImplementedException();
		}

		// Token: 0x060008D2 RID: 2258 RVA: 0x0002AE28 File Offset: 0x00029028
		public void SaveBinding()
		{
			if (this.SpecBinding != null)
			{
				SpecBindingInfo specBindingInfo = new SpecBindingInfo();
				specBindingInfo.ObjectBindings = new ObjectBindings();
				specBindingInfo.ObjectBindings.bindings = this.SpecBinding.ToArray();
				XmlSerializer xmlSerializer = new XmlSerializer(typeof(SpecBindingInfo));
				using (TextWriter textWriter = new StringWriter())
				{
					xmlSerializer.Serialize(textWriter, specBindingInfo);
					this._tzpManager.ElementBindings = textWriter.ToString();
				}
			}
		}

		// Token: 0x040002F4 RID: 756
		public static readonly string[] AllowSaveWidgets = new string[]
		{
			"ButtonEdit", "CheckBox", "ComboBox", "DateEdit", "Edit", "Phantom", "FFImage", "FFLabel", "ProgressBar", "RadioGroup",
			"Slider", "SpinEdit", "TextEdit", "TimeEdit", "WebComponent", "DateTimeEdit"
		};

		// Token: 0x040002F5 RID: 757
		private static string[] DisabledActions = new string[] { "home", "logistics", "personalwork", "agendum", "locale", "qbe_select", "qbe_save", "controlr", "help", "exit" };

		// Token: 0x040002F6 RID: 758
		private FormSpecDictionary _formSpecDic;

		// Token: 0x040002F7 RID: 759
		private TzpManager _tzpManager;

		// Token: 0x040002F8 RID: 760
		public string[] AllowedAction;

		// Token: 0x040002F9 RID: 761
		private BackgroundWorker _TSDValidater;

		// Token: 0x040002FA RID: 762
		private BackgroundWorker _FormValidater;

		// Token: 0x040002FB RID: 763
		private List<XElement> excludes = new List<XElement>();

		// Token: 0x040002FC RID: 764
		private List<SpecProgRelNode> prog_rels = new List<SpecProgRelNode>();

		// Token: 0x040002FD RID: 765
		private List<SpecReferenceNode> ref_fields = new List<SpecReferenceNode>();

		// Token: 0x040002FE RID: 766
		private List<SpecMultiLangNode> multi_langs = new List<SpecMultiLangNode>();

		// Token: 0x040002FF RID: 767
		private List<SpecHelpCodeNode> help_codes = new List<SpecHelpCodeNode>();

		// Token: 0x04000300 RID: 768
		private List<SpecFieldNode> fields = new List<SpecFieldNode>();

		// Token: 0x04000301 RID: 769
		private ObservableCollection<SpecActionNode> _acts = new ObservableCollection<SpecActionNode>();

		// Token: 0x04000302 RID: 770
		private ReadOnlyObservableCollection<SpecActionNode> _readonyActs;

		// Token: 0x04000303 RID: 771
		private List<SpecTreeNode> trees = new List<SpecTreeNode>();

		// Token: 0x04000304 RID: 772
		private List<SpecActStringNode> actStrings = new List<SpecActStringNode>();

		// Token: 0x04000305 RID: 773
		private List<SpecFieldStringNode> fieldStrings = new List<SpecFieldStringNode>();

		// Token: 0x04000306 RID: 774
		private ScreenRecordManager _screenRecordManager;
	}
}
