using System;
using System.CodeDom.Compiler;
using System.Collections.Generic;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using Microsoft.Practices.Prism.Events;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.Output
{
	// Token: 0x02000005 RID: 5
	public partial class OutputPanelControl : UserControl, INotifyPropertyChanged
	{
		// Token: 0x06000009 RID: 9 RVA: 0x000020D4 File Offset: 0x000002D4
		public OutputPanelControl()
		{
			this.InitializeComponent();
			if (DesignerProperties.GetIsInDesignMode(this))
			{
				return;
			}
			EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>().Subscribe(new Action<DocumentErrorsEventArgs>(this.Subscribe_ReceiveDocumentErrors), ThreadOption.BackgroundThread);
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.Subscribe_TzpClosed));
			this._errors = base.Resources["errors"] as DocumentErrorsSource;
		}

		// Token: 0x0600000A RID: 10 RVA: 0x00002168 File Offset: 0x00000368
		public void Subscribe_TzpClosed(PackageKey key)
		{
			List<DocumentErrorsEventArgs> list = this._errors.Where<DocumentErrorsEventArgs>((DocumentErrorsEventArgs args) => key.Equals(args.ProgramKey)).ToList<DocumentErrorsEventArgs>();
			foreach (DocumentErrorsEventArgs e in list)
			{
				this._errors.Remove(e);
			}
		}

		// Token: 0x17000004 RID: 4
		// (get) Token: 0x0600000B RID: 11 RVA: 0x000021E8 File Offset: 0x000003E8
		// (set) Token: 0x0600000C RID: 12 RVA: 0x000021FA File Offset: 0x000003FA
		public bool IsActive
		{
			get
			{
				return (bool)base.GetValue(OutputPanelControl.IsActiveProperty);
			}
			set
			{
				base.SetValue(OutputPanelControl.IsActiveProperty, value);
				this.NotifyPropertyChanged("IsActive");
			}
		}

		// Token: 0x0600000D RID: 13 RVA: 0x000022D0 File Offset: 0x000004D0
		public void Subscribe_ReceiveDocumentErrors(DocumentErrorsEventArgs args)
		{
			base.Dispatcher.BeginInvoke(new Action(delegate
			{
				ErrorComparer comparer = new ErrorComparer();
				DocumentErrorsEventArgs e2 = this._errors.Where<DocumentErrorsEventArgs>((DocumentErrorsEventArgs e) => comparer.Equals(args, e)).ElementAtOrDefault<DocumentErrorsEventArgs>(0);
				this._errors.Remove(e2);
				this._errors.Add(args);
				if (args.ErrorType == ErrorsType.ERROR)
				{
					this.IsActive = true;
				}
			}), new object[0]);
		}

		// Token: 0x0600000E RID: 14 RVA: 0x0000230F File Offset: 0x0000050F
		private void ToggleButton_Click(object sender, RoutedEventArgs e)
		{
			CollectionViewSource.GetDefaultView(this.dataGrid1.ItemsSource).Refresh();
		}

		// Token: 0x0600000F RID: 15 RVA: 0x00002326 File Offset: 0x00000526
		public void NotifyPropertyChanged(string propertyName)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(propertyName));
			}
		}

		// Token: 0x14000001 RID: 1
		// (add) Token: 0x06000010 RID: 16 RVA: 0x00002344 File Offset: 0x00000544
		// (remove) Token: 0x06000011 RID: 17 RVA: 0x0000237C File Offset: 0x0000057C
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000012 RID: 18 RVA: 0x000023B4 File Offset: 0x000005B4
		private void DataGrid_MouseDoubleClick(object sender, MouseButtonEventArgs e)
		{
			if (sender is DataGrid)
			{
				DataGrid dataGrid = sender as DataGrid;
				if (dataGrid.SelectedItems != null)
				{
					DocumentErrorsEventArgs e2 = dataGrid.SelectedItem as DocumentErrorsEventArgs;
					if (e2 != null)
					{
						if (string.IsNullOrEmpty(e2.Key))
						{
							return;
						}
						SearchResultInfo searchResultInfo = new SearchResultInfo();
						searchResultInfo.Key = e2.Key;
						searchResultInfo.ProgramKey = e2.ProgramKey;
						searchResultInfo.SourceType = e2.SourceType;
						EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Publish(searchResultInfo);
					}
				}
			}
		}

		// Token: 0x06000013 RID: 19 RVA: 0x00002430 File Offset: 0x00000630
		private bool filterErrorType(DocumentErrorsEventArgs args)
		{
			bool flag = false;
			switch (args.ErrorType)
			{
			case ErrorsType.ERROR:
				flag = this.btnError.IsChecked ?? false;
				break;
			case ErrorsType.WARNING:
				flag = this.btnWarning.IsChecked ?? false;
				break;
			case ErrorsType.INFORMATION:
				flag = this.btnInfo.IsChecked ?? false;
				break;
			}
			return flag;
		}

		// Token: 0x06000014 RID: 20 RVA: 0x000024C0 File Offset: 0x000006C0
		private void CollectionViewSource_Filter(object sender, FilterEventArgs e)
		{
			DocumentErrorsEventArgs e2 = e.Item as DocumentErrorsEventArgs;
			e.Accepted = this.filterErrorType(e2);
		}

		// Token: 0x06000015 RID: 21 RVA: 0x000024E6 File Offset: 0x000006E6
		private void Button_Click(object sender, RoutedEventArgs e)
		{
			this._errors.Clear();
		}

		// Token: 0x04000004 RID: 4
		private DocumentErrorsSource _errors;

		// Token: 0x04000005 RID: 5
		public static readonly DependencyProperty IsActiveProperty = DependencyProperty.Register("IsActive", typeof(bool), typeof(OutputPanelControl), new PropertyMetadata(true));
	}
}
