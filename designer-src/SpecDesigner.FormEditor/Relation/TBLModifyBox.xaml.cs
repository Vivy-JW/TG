using System;
using System.CodeDom.Compiler;
using System.Collections;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Markup;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Relation
{
	// Token: 0x02000025 RID: 37
	public partial class TBLModifyBox : UserControl
	{
		// Token: 0x1700003A RID: 58
		// (get) Token: 0x0600012F RID: 303 RVA: 0x00006F3A File Offset: 0x0000513A
		// (set) Token: 0x06000130 RID: 304 RVA: 0x00006F4C File Offset: 0x0000514C
		public IEnumerable Tables
		{
			get
			{
				return (IEnumerable)base.GetValue(TBLModifyBox.TablesProperty);
			}
			set
			{
				base.SetValue(TBLModifyBox.TablesProperty, value);
			}
		}

		// Token: 0x06000131 RID: 305 RVA: 0x00006F5A File Offset: 0x0000515A
		public TBLModifyBox()
		{
			this.InitializeComponent();
		}

		// Token: 0x06000132 RID: 306 RVA: 0x00006F73 File Offset: 0x00005173
		public void Show(TBLModel tBLModel)
		{
			this._realModel = tBLModel;
			this.UpdateUpperTableList(tBLModel);
			this._tempTBLModel = TBLModifyBox.TBLModelForCheck.Create(tBLModel);
			base.Visibility = Visibility.Visible;
			base.DataContext = this._tempTBLModel;
		}

		// Token: 0x1700003B RID: 59
		// (get) Token: 0x06000133 RID: 307 RVA: 0x00006FA2 File Offset: 0x000051A2
		// (set) Token: 0x06000134 RID: 308 RVA: 0x00006FAA File Offset: 0x000051AA
		public ObservableCollection<TableComboBoxItem> UpperTables
		{
			get
			{
				return this._upperTables;
			}
			private set
			{
				value = this._upperTables;
			}
		}

		// Token: 0x06000135 RID: 309 RVA: 0x00006FB4 File Offset: 0x000051B4
		private void UpdateUpperTableList(TBLModel tBLModel)
		{
			if (this.UpperTables == null)
			{
				this.UpperTables = new ObservableCollection<TableComboBoxItem>();
			}
			else
			{
				this.UpperTables.Clear();
			}
			this.UpperTables.Add(new TableComboBoxItem
			{
				Name = "",
				Text = "------",
				IsEnabled = true
			});
			TableAssociationModel associateTable = SettingManager.Get().GetTzpManger(tBLModel.Key).SpecificationInfo.AssociateTable;
			foreach (TBLModel tblmodel in associateTable.AliveTBLs)
			{
				this.UpperTables.Add(new TableComboBoxItem
				{
					Name = tblmodel.TBLName,
					Text = string.Format("{0} {1}", tblmodel.TBLName, TableColumnHelper.GetTableDesc(tblmodel.TBLName)),
					IsEnabled = (tblmodel != tBLModel && tblmodel.UpperTable != tBLModel.TBLName)
				});
			}
		}

		// Token: 0x06000136 RID: 310 RVA: 0x000070C8 File Offset: 0x000052C8
		private void CancelTBLBox_MouseLeftButtonUp(object sender, RoutedEventArgs e)
		{
			e.Handled = true;
			base.Visibility = Visibility.Collapsed;
			base.DataContext = Binding.DoNothing;
		}

		// Token: 0x06000137 RID: 311 RVA: 0x000070E4 File Offset: 0x000052E4
		private void modifyUpperTBLNameCB_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			TBLModel tblmodel = base.DataContext as TBLModel;
			if (tblmodel == null)
			{
				return;
			}
			string text = this.modifyUpperTBLNameCB.SelectedValue as string;
			if (!string.IsNullOrEmpty(text))
			{
				this.modifyUpperKeyTB.Text = TableColumnHelper.GetPK(text);
				if (string.IsNullOrEmpty(this.modifyThisKeyTB.Text))
				{
					this.modifyThisKeyTB.Text = TableColumnHelper.GetPK(tblmodel.TBLName);
				}
			}
			else
			{
				this.modifyUpperKeyTB.Text = (this.modifyThisKeyTB.Text = string.Empty);
			}
			this.modifyUpperKeyTB.IsEnabled = (this.modifyThisKeyTB.IsEnabled = !string.IsNullOrEmpty(text));
		}

		// Token: 0x06000138 RID: 312 RVA: 0x00007198 File Offset: 0x00005398
		private void modifyTBLNameCB_SelectionChanged(object sender, SelectionChangedEventArgs e)
		{
			if (!(base.DataContext is TBLModel))
			{
				return;
			}
			string text = this.modifyTBLNameCB.SelectedValue as string;
			if (!string.IsNullOrEmpty(text) && !string.IsNullOrEmpty(this.modifyUpperTBLNameCB.SelectedValue as string))
			{
				this.modifyThisKeyTB.Text = TableColumnHelper.GetPK(text);
			}
		}

		// Token: 0x1700003C RID: 60
		// (get) Token: 0x06000139 RID: 313 RVA: 0x00007206 File Offset: 0x00005406
		public RelayCommand ConfirmCommand
		{
			get
			{
				if (this._confirmCommand == null)
				{
					this._confirmCommand = new RelayCommand(delegate(object p)
					{
						this.ExecuteConfirm();
					}, (object p) => this.CanExecuteConfirm());
				}
				return this._confirmCommand;
			}
		}

		// Token: 0x0600013A RID: 314 RVA: 0x00007239 File Offset: 0x00005439
		private bool CanExecuteConfirm()
		{
			return TBLModifyBox.IsValid(this);
		}

		// Token: 0x0600013B RID: 315 RVA: 0x00007244 File Offset: 0x00005444
		private void ExecuteConfirm()
		{
			this._realModel.TBLName = this._tempTBLModel.TBLName;
			this._realModel.PK = this._tempTBLModel.PK;
			this._realModel.FkDetail = this._tempTBLModel.FkDetail;
			this._realModel.FkMaster = this._tempTBLModel.FkMaster;
			this._realModel.UpperKey = this._tempTBLModel.UpperKey;
			this._realModel.UpperTable = this._tempTBLModel.UpperTable;
			this._realModel.ThisKey = this._tempTBLModel.ThisKey;
			base.Visibility = Visibility.Collapsed;
			base.DataContext = Binding.DoNothing;
		}

		// Token: 0x0600013C RID: 316 RVA: 0x00007300 File Offset: 0x00005500
		public static bool IsValid(DependencyObject parent)
		{
			bool flag = true;
			LocalValueEnumerator localValueEnumerator = parent.GetLocalValueEnumerator();
			while (localValueEnumerator.MoveNext())
			{
				LocalValueEntry localValueEntry = localValueEnumerator.Current;
				if (BindingOperations.IsDataBound(parent, localValueEntry.Property))
				{
					BindingOperations.GetBinding(parent, localValueEntry.Property);
					BindingExpression bindingExpression = BindingOperations.GetBindingExpression(parent, localValueEntry.Property);
					bindingExpression.UpdateSource();
					if (bindingExpression.HasError)
					{
						flag = false;
					}
				}
			}
			IEnumerable children = LogicalTreeHelper.GetChildren(parent);
			foreach (object obj in children)
			{
				if (obj is DependencyObject)
				{
					DependencyObject dependencyObject = (DependencyObject)obj;
					if (!TBLModifyBox.IsValid(dependencyObject))
					{
						flag = false;
					}
				}
			}
			return flag;
		}

		// Token: 0x040000A6 RID: 166
		public static readonly DependencyProperty TablesProperty = DependencyProperty.Register("Tables", typeof(IEnumerable), typeof(TBLModifyBox), new FrameworkPropertyMetadata(null));

		// Token: 0x040000A7 RID: 167
		private TBLModifyBox.TBLModelForCheck _tempTBLModel;

		// Token: 0x040000A8 RID: 168
		private TBLModel _realModel;

		// Token: 0x040000A9 RID: 169
		private ObservableCollection<TableComboBoxItem> _upperTables = new ObservableCollection<TableComboBoxItem>();

		// Token: 0x040000AA RID: 170
		private RelayCommand _confirmCommand;

		// Token: 0x02000026 RID: 38
		private class TBLModelForCheck : INotifyPropertyChanged, IDataErrorInfo
		{
			// Token: 0x06000142 RID: 322 RVA: 0x00007520 File Offset: 0x00005720
			public static TBLModifyBox.TBLModelForCheck Create(TBLModel model)
			{
				return new TBLModifyBox.TBLModelForCheck
				{
					PK = model.PK,
					TBLName = model.TBLName,
					_fkMaster = model.FkMaster,
					_fkDetail = model.FkDetail,
					UpperKey = model.UpperKey,
					UpperTable = model.UpperTable,
					ThisKey = model.ThisKey,
					Main = model.Main
				};
			}

			// Token: 0x1700003D RID: 61
			// (get) Token: 0x06000144 RID: 324 RVA: 0x000075A7 File Offset: 0x000057A7
			// (set) Token: 0x06000145 RID: 325 RVA: 0x000075AF File Offset: 0x000057AF
			public string TBLName
			{
				get
				{
					return this._tblName;
				}
				set
				{
					this._tblName = value;
					this.OnPropertyChanged("TBLName");
				}
			}

			// Token: 0x1700003E RID: 62
			// (get) Token: 0x06000146 RID: 326 RVA: 0x000075C3 File Offset: 0x000057C3
			// (set) Token: 0x06000147 RID: 327 RVA: 0x000075CB File Offset: 0x000057CB
			public string PK
			{
				get
				{
					return this._pk;
				}
				set
				{
					this._pk = value;
					this.OnPropertyChanged("PK");
				}
			}

			// Token: 0x1700003F RID: 63
			// (get) Token: 0x06000148 RID: 328 RVA: 0x000075DF File Offset: 0x000057DF
			// (set) Token: 0x06000149 RID: 329 RVA: 0x000075E7 File Offset: 0x000057E7
			public string FkMaster
			{
				get
				{
					return this._fkMaster;
				}
				set
				{
					if (Regex.IsMatch(value, TBLModifyBox.TBLModelForCheck.ColumnCheckPattern))
					{
						throw new Exception("不允許常數");
					}
					this._fkMaster = value;
					this.OnPropertyChanged("FKMaster");
				}
			}

			// Token: 0x17000040 RID: 64
			// (get) Token: 0x0600014A RID: 330 RVA: 0x00007613 File Offset: 0x00005813
			// (set) Token: 0x0600014B RID: 331 RVA: 0x0000761B File Offset: 0x0000581B
			public string FkDetail
			{
				get
				{
					return this._fkDetail;
				}
				set
				{
					if (Regex.IsMatch(value, TBLModifyBox.TBLModelForCheck.ColumnCheckPattern))
					{
						throw new Exception("不允許常數");
					}
					this._fkDetail = value;
					this.OnPropertyChanged("FkDetail");
				}
			}

			// Token: 0x17000041 RID: 65
			// (get) Token: 0x0600014C RID: 332 RVA: 0x00007647 File Offset: 0x00005847
			// (set) Token: 0x0600014D RID: 333 RVA: 0x0000764F File Offset: 0x0000584F
			public string UpperTable
			{
				get
				{
					return this._upperTable;
				}
				set
				{
					this._upperTable = value;
					this.OnPropertyChanged("UpperTable");
				}
			}

			// Token: 0x17000042 RID: 66
			// (get) Token: 0x0600014E RID: 334 RVA: 0x00007663 File Offset: 0x00005863
			// (set) Token: 0x0600014F RID: 335 RVA: 0x0000766B File Offset: 0x0000586B
			public string UpperKey
			{
				get
				{
					return this._upperKey;
				}
				set
				{
					this._upperKey = value;
					this.OnPropertyChanged("UpperKey");
				}
			}

			// Token: 0x17000043 RID: 67
			// (get) Token: 0x06000150 RID: 336 RVA: 0x0000767F File Offset: 0x0000587F
			// (set) Token: 0x06000151 RID: 337 RVA: 0x00007687 File Offset: 0x00005887
			public string ThisKey
			{
				get
				{
					return this._thisKey;
				}
				set
				{
					this._thisKey = value;
					this.OnPropertyChanged("ThisKey");
				}
			}

			// Token: 0x17000044 RID: 68
			// (get) Token: 0x06000152 RID: 338 RVA: 0x0000769B File Offset: 0x0000589B
			// (set) Token: 0x06000153 RID: 339 RVA: 0x000076A3 File Offset: 0x000058A3
			public string Main { get; set; }

			// Token: 0x14000003 RID: 3
			// (add) Token: 0x06000154 RID: 340 RVA: 0x000076AC File Offset: 0x000058AC
			// (remove) Token: 0x06000155 RID: 341 RVA: 0x000076E4 File Offset: 0x000058E4
			public event PropertyChangedEventHandler PropertyChanged;

			// Token: 0x06000156 RID: 342 RVA: 0x00007719 File Offset: 0x00005919
			private void OnPropertyChanged(string propertyName)
			{
				if (this.PropertyChanged != null)
				{
					this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
				}
			}

			// Token: 0x17000045 RID: 69
			// (get) Token: 0x06000157 RID: 343 RVA: 0x00007735 File Offset: 0x00005935
			public string Error
			{
				get
				{
					return string.Empty;
				}
			}

			// Token: 0x17000046 RID: 70
			public string this[string columnName]
			{
				get
				{
					string text = string.Empty;
					if (columnName != null && (columnName == "FkMaster" || columnName == "FkDetail"))
					{
						if ((this.FkMaster.Length != 0 || this.FkDetail.Length != 0) && this.FkMaster.Length != this.FkDetail.Length)
						{
							text = "FKMaster 與 FkDetail 設定數量不同";
						}
						else if (this.FkMaster.Split(new char[] { ',' }).Count<string>() != this.FkDetail.Split(new char[] { ',' }).Count<string>())
						{
							text = "FKMaster 與 FkDetail 設定數量不同";
						}
					}
					return text;
				}
			}

			// Token: 0x040000B4 RID: 180
			private static readonly string ColumnCheckPattern = "[\"']";

			// Token: 0x040000B5 RID: 181
			private string _tblName;

			// Token: 0x040000B6 RID: 182
			private string _pk;

			// Token: 0x040000B7 RID: 183
			private string _fkMaster;

			// Token: 0x040000B8 RID: 184
			private string _fkDetail = string.Empty;

			// Token: 0x040000B9 RID: 185
			private string _upperTable;

			// Token: 0x040000BA RID: 186
			private string _upperKey;

			// Token: 0x040000BB RID: 187
			private string _thisKey;
		}
	}
}
