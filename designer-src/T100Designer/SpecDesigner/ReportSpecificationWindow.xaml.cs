using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner
{
	// Token: 0x02000038 RID: 56
	public partial class ReportSpecificationWindow : UserControl, IDisposable
	{
		// Token: 0x060002B9 RID: 697 RVA: 0x0000CA33 File Offset: 0x0000AC33
		public ReportSpecificationWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x060002BA RID: 698 RVA: 0x0000CA41 File Offset: 0x0000AC41
		public ReportSpecificationWindow(PackageKey key)
			: this()
		{
			this.ProgramKey = key;
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingReportSpecEvent>().Subscribe(new Action<PackageKey>(this.LoadFile));
		}

		// Token: 0x060002BB RID: 699 RVA: 0x0000CA6C File Offset: 0x0000AC6C
		private void LoadFile(PackageKey key)
		{
			if (this.ProgramKey != key)
			{
				return;
			}
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingReportSpecEvent>().Unsubscribe(new Action<PackageKey>(this.LoadFile));
			base.DataContext = SettingManager.Get().GetTzpManger(this.ProgramKey).ReportSpecificationInfo;
			this.notBookingWatermark.Visibility = (SettingManager.Get().GetTzpManger(this.ProgramKey).Booking ? Visibility.Collapsed : Visibility.Visible);
			EventAggregatorManager.Global.GetEvent<SaveSettingEvent>().Subscribe(new Action<PackageKey>(this.SaveContent));
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Subscribe(new Action<PackageKey>(this.OnTzpFileClose));
		}

		// Token: 0x060002BC RID: 700 RVA: 0x0000CB1C File Offset: 0x0000AD1C
		private void SaveContent(PackageKey key)
		{
			if (this.ProgramKey != key)
			{
				return;
			}
			SettingManager.Get().GetTzpManger(this.ProgramKey).SaveSpecificationForReport();
		}

		// Token: 0x060002BD RID: 701 RVA: 0x0000CB42 File Offset: 0x0000AD42
		private void OnTzpFileClose(PackageKey key)
		{
			if (key == this.ProgramKey)
			{
				this.Dispose();
			}
		}

		// Token: 0x060002BE RID: 702 RVA: 0x0000CB58 File Offset: 0x0000AD58
		public void Dispose()
		{
			EventAggregatorManager.Global.GetEvent<TzpFileClose>().Unsubscribe(new Action<PackageKey>(this.OnTzpFileClose));
			EventAggregatorManager.Global.GetEvent<SaveSettingEvent>().Unsubscribe(new Action<PackageKey>(this.SaveContent));
			base.DataContext = null;
		}

		// Token: 0x0400018C RID: 396
		private PackageKey ProgramKey;
	}
}
