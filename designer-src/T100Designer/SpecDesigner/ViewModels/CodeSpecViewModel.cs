using System;
using System.Windows.Media.Imaging;
using SpecDesigner.CodeEditWindow.View;
using SpecDesignerCommon;
using SpecDesignerCommon.Events;

namespace SpecDesigner.ViewModels
{
	// Token: 0x02000028 RID: 40
	public class CodeSpecViewModel : FileViewModel
	{
		// Token: 0x06000253 RID: 595 RVA: 0x0000AE98 File Offset: 0x00009098
		public CodeSpecViewModel(PackageKey key)
			: base(key)
		{
			base.UI = new CodeSpecificationMainWindow(key);
			base.IconSource = new BitmapImage(new Uri("/Images/document_spec.png", UriKind.Relative));
			EventAggregatorManager.Global.GetEvent<LoadedSettingIncludingCodeSpecEvent>().Publish(key);
			EventAggregatorManager.Get(key).GetEvent<CodeSpecChangedEvent>().Subscribe(new Action<PackageKey>(this.OnCodeSpecChanged));
		}

		// Token: 0x06000254 RID: 596 RVA: 0x0000AEFB File Offset: 0x000090FB
		public override bool CanClose()
		{
			return true;
		}

		// Token: 0x06000255 RID: 597 RVA: 0x0000AEFE File Offset: 0x000090FE
		public override void OnClose()
		{
			if (EditorWorkspace.This.Close(this))
			{
				EventAggregatorManager.Get(base.Key).GetEvent<CodeChangedEvent>().Unsubscribe(new Action<PackageKey>(this.OnCodeSpecChanged));
				base.OnClose();
			}
			GC.Collect();
		}

		// Token: 0x06000256 RID: 598 RVA: 0x0000AF39 File Offset: 0x00009139
		private void OnCodeSpecChanged(PackageKey key)
		{
			if (base.Key == key)
			{
				base.IsModified = true;
			}
		}
	}
}
