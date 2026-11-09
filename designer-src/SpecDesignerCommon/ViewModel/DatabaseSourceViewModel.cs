using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Linq;
using System.Windows;
using System.Xml.Linq;
using SpecDesigner.Controls.Controls;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.Logger;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000092 RID: 146
	public class DatabaseSourceViewModel : INotifyPropertyChanged
	{
		// Token: 0x14000016 RID: 22
		// (add) Token: 0x060005EC RID: 1516 RVA: 0x0001B198 File Offset: 0x00019398
		// (remove) Token: 0x060005ED RID: 1517 RVA: 0x0001B1D0 File Offset: 0x000193D0
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x170001B0 RID: 432
		// (get) Token: 0x060005EE RID: 1518 RVA: 0x0001B205 File Offset: 0x00019405
		// (set) Token: 0x060005EF RID: 1519 RVA: 0x0001B20D File Offset: 0x0001940D
		public List<DSTableModel> Tables { get; set; }

		// Token: 0x060005F0 RID: 1520 RVA: 0x0001B216 File Offset: 0x00019416
		public DatabaseSourceViewModel()
		{
			this.Tables = new List<DSTableModel>();
		}

		// Token: 0x060005F1 RID: 1521 RVA: 0x0001B308 File Offset: 0x00019508
		public DatabaseSourceViewModel(PackageKey programKey, TableAssociationModel tableAssociation, string env, FormSpecDictionary dict)
			: this()
		{
			List<FormSpecModel> list = dict.Values.Where<FormSpecModel>((FormSpecModel x) => x.SpecField != null).ToList<FormSpecModel>();
			list = list.Where<FormSpecModel>((FormSpecModel x) => x.GeneroComponent.Parent.Name != "s_queryplan" && x.GeneroComponent.Parent.Name != "s_relateapps" && x.GeneroComponent.Parent.Name != "s_browse").ToList<FormSpecModel>();
			string text = "";
			bool flag = false;
			TBLModel tbl;
			foreach (TBLModel tblmodel in tableAssociation.AliveTBLs)
			{
				tbl = tblmodel;
				try
				{
					DSTableModel dstableModel = new DSTableModel
					{
						Table = tbl
					};
					XElement ele;
					foreach (XElement xelement in TableColumnHelper.FindTableColumns(tbl.TBLName, env))
					{
						ele = xelement;
						DSColumnViewModel dscolumnViewModel = new DSColumnViewModel();
						dscolumnViewModel.TBLName = tbl.TBLName;
						dscolumnViewModel.Name = ele.Attribute("name").Value;
						dscolumnViewModel.Description = ele.Attribute("text").Value;
						dscolumnViewModel.Detail = ele;
						dscolumnViewModel.IsUsed = list.Any<FormSpecModel>((FormSpecModel x) => x.SpecField.Table == tbl.TBLName && x.SpecField.Column == ele.Attribute("name").Value);
						dstableModel.Columns.Add(dscolumnViewModel);
					}
					if (tbl.Main == "Y")
					{
						this.Tables.Insert(0, dstableModel);
					}
					else
					{
						this.Tables.Add(dstableModel);
					}
				}
				catch (Exception)
				{
					flag = true;
					text += (string.IsNullOrEmpty(text) ? tbl.TBLName : ("," + tbl.TBLName));
					string text2 = string.Format(Application.Current.FindResource("Message_TableNotFound") as string, programKey.Program, tbl.TBLName);
					DSCLogger.Write(text2, programKey.Program);
				}
			}
			if (flag)
			{
				DesignerMessageBox.Show(string.Format(Application.Current.FindResource("Message_TableNotFound") as string, programKey.Program, text), Application.Current.FindResource("Message_Warning") as string, MessageBoxButton.OK, MessageBoxImage.Exclamation);
			}
		}

		// Token: 0x170001B1 RID: 433
		// (get) Token: 0x060005F2 RID: 1522 RVA: 0x0001B600 File Offset: 0x00019800
		public static DatabaseSourceViewModel This
		{
			get
			{
				if (DatabaseSourceViewModel._this == null)
				{
					DatabaseSourceViewModel._this = new DatabaseSourceViewModel();
				}
				return DatabaseSourceViewModel._this;
			}
		}

		// Token: 0x170001B2 RID: 434
		// (get) Token: 0x060005F3 RID: 1523 RVA: 0x0001B618 File Offset: 0x00019818
		public SpecificationInfo SpecificationInfo
		{
			get
			{
				if (!(null == this._programKey))
				{
					return SettingManager.Get().GetTzpManger(this._programKey).SpecificationInfo;
				}
				return null;
			}
		}

		// Token: 0x170001B3 RID: 435
		// (get) Token: 0x060005F4 RID: 1524 RVA: 0x0001B63F File Offset: 0x0001983F
		public DatabaseSourceViewModel DatabaseSource
		{
			get
			{
				return this.SpecificationInfo.DatabaseSource;
			}
		}

		// Token: 0x060005F5 RID: 1525 RVA: 0x0001B64C File Offset: 0x0001984C
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x060005F6 RID: 1526 RVA: 0x0001B720 File Offset: 0x00019920
		internal void RefreshUsed(FormSpecDictionary dict)
		{
			List<FormSpecModel> list = dict.Values.Where<FormSpecModel>((FormSpecModel x) => x.SpecField != null).ToList<FormSpecModel>();
			list = list.Where<FormSpecModel>((FormSpecModel x) => x.GeneroComponent.Parent.Name != "s_queryplan" && x.GeneroComponent.Parent.Name != "s_relateapps" && x.GeneroComponent.Parent.Name != "s_browse").ToList<FormSpecModel>();
			foreach (DSTableModel dstableModel in this.Tables)
			{
				DSColumnViewModel cModel;
				foreach (DSColumnViewModel dscolumnViewModel in dstableModel.Columns)
				{
					cModel = dscolumnViewModel;
					cModel.IsUsed = list.Any<FormSpecModel>((FormSpecModel x) => x.SpecField.Table == cModel.TBLName && x.SpecField.Column == cModel.Name);
				}
			}
		}

		// Token: 0x170001B4 RID: 436
		// (get) Token: 0x060005F7 RID: 1527 RVA: 0x0001B830 File Offset: 0x00019A30
		// (set) Token: 0x060005F8 RID: 1528 RVA: 0x0001B838 File Offset: 0x00019A38
		public PackageKey ProgramKey
		{
			get
			{
				return this._programKey;
			}
			set
			{
				if (this._programKey == value)
				{
					return;
				}
				this._programKey = value;
				this.NotifyPropertyChanged("ProgramKey");
				this.NotifyPropertyChanged("SpecificationInfo");
				this.NotifyPropertyChanged("DatabaseSource");
			}
		}

		// Token: 0x04000243 RID: 579
		private PackageKey _programKey;

		// Token: 0x04000245 RID: 581
		private static DatabaseSourceViewModel _this;
	}
}
