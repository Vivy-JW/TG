using System;
using System.Windows.Media.Imaging;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.ViewModels
{
	// Token: 0x02000014 RID: 20
	public class ReportSpecViewModel : FileViewModel
	{
		// Token: 0x060001DC RID: 476 RVA: 0x00008140 File Offset: 0x00006340
		public ReportSpecViewModel(PackageKey key)
			: base(key)
		{
			base.UI = new ReportSpecificationWindow(key);
			base.IconSource = new BitmapImage(new Uri("/Images/document_spec.png", UriKind.Relative));
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingReportSpecEvent>().Publish(key);
			EventAggregatorManager.Get(key).GetEvent<ReportSpecChangedEvent>().Subscribe(new Action<PackageKey>(this.OnReportSpecChanged));
		}

		// Token: 0x060001DD RID: 477 RVA: 0x000081A3 File Offset: 0x000063A3
		public override void OnClose()
		{
			if (EditorWorkspace.This.Close(this))
			{
				EventAggregatorManager.Get(base.Key).GetEvent<ReportSpecChangedEvent>().Unsubscribe(new Action<PackageKey>(this.OnReportSpecChanged));
				base.OnClose();
			}
			GC.Collect();
		}

		// Token: 0x060001DE RID: 478 RVA: 0x000081DE File Offset: 0x000063DE
		private void OnReportSpecChanged(PackageKey key)
		{
			if (base.Key == key)
			{
				base.IsModified = true;
			}
		}
	}
}
