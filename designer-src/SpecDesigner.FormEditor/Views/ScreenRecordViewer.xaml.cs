using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Markup;
using System.Xml.Linq;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x02000022 RID: 34
	public partial class ScreenRecordViewer : UserControl, INotifyPropertyChanged
	{
		// Token: 0x17000038 RID: 56
		// (get) Token: 0x0600011B RID: 283 RVA: 0x00006AF1 File Offset: 0x00004CF1
		public static ScreenRecordViewer This
		{
			get
			{
				if (ScreenRecordViewer._this == null)
				{
					ScreenRecordViewer._this = new ScreenRecordViewer();
				}
				return ScreenRecordViewer._this;
			}
		}

		// Token: 0x0600011C RID: 284 RVA: 0x00006B09 File Offset: 0x00004D09
		private ScreenRecordViewer()
		{
			this.InitializeComponent();
		}

		// Token: 0x0600011D RID: 285 RVA: 0x00006B17 File Offset: 0x00004D17
		private SpecificationInfo GetSpecificationInfo()
		{
			if (!(null == this._programKey))
			{
				return SettingManager.Get().GetTzpManger(this._programKey).SpecificationInfo;
			}
			return null;
		}

		// Token: 0x17000039 RID: 57
		// (get) Token: 0x0600011E RID: 286 RVA: 0x00006B3E File Offset: 0x00004D3E
		// (set) Token: 0x0600011F RID: 287 RVA: 0x00006B48 File Offset: 0x00004D48
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
				base.DataContext = ((this._programKey != null) ? this.GetSpecificationInfo().GetScreenRecords() : Binding.DoNothing);
				this.UITree.DataContext = ((this._programKey != null) ? this.GetSpecificationInfo().FormNode : Binding.DoNothing);
				XmlElement.ExpandCurrentChild(this.UITree.DataContext as XmlElement, true);
				this.NotifyPropertyChanged("ScreenRecords");
				this.NotifyPropertyChanged("ProgramKey");
			}
		}

		// Token: 0x14000002 RID: 2
		// (add) Token: 0x06000120 RID: 288 RVA: 0x00006BE8 File Offset: 0x00004DE8
		// (remove) Token: 0x06000121 RID: 289 RVA: 0x00006C20 File Offset: 0x00004E20
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x06000122 RID: 290 RVA: 0x00006C55 File Offset: 0x00004E55
		public void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x06000123 RID: 291 RVA: 0x00006C74 File Offset: 0x00004E74
		private void ScreenRecord_SelectedItemChanged(object sender, RoutedPropertyChangedEventArgs<object> e)
		{
			XElement xelement = e.NewValue as XElement;
			if (xelement == null)
			{
				return;
			}
			SearchResultInfo searchResultInfo = new SearchResultInfo();
			searchResultInfo.Key = xelement.Attribute("name").Value;
			searchResultInfo.ProgramKey = this.ProgramKey;
			searchResultInfo.SourceType = this.ProgramKey.PackType;
			EventAggregatorManager.Global.GetEvent<SearchResultInfoSelectedEvent>().Publish(searchResultInfo);
		}

		// Token: 0x06000124 RID: 292 RVA: 0x00006CDF File Offset: 0x00004EDF
		private void ScreenRecordRefresh_Click(object sender, RoutedEventArgs e)
		{
			base.DataContext = ((this.ProgramKey != null) ? this.GetSpecificationInfo().GetScreenRecords() : Binding.DoNothing);
		}

		// Token: 0x06000125 RID: 293 RVA: 0x00006D07 File Offset: 0x00004F07
		private void FormStructureRefresh_Click(object sender, RoutedEventArgs e)
		{
			this.UITree.DataContext = Binding.DoNothing;
			this.UITree.DataContext = ((this.ProgramKey != null) ? this.GetSpecificationInfo().FormNode : Binding.DoNothing);
		}

		// Token: 0x040000A0 RID: 160
		private static ScreenRecordViewer _this;

		// Token: 0x040000A1 RID: 161
		private PackageKey _programKey;
	}
}
