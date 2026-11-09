using System;
using System.Windows;

namespace SpecDesigner.ViewModels
{
	// Token: 0x0200002C RID: 44
	public class SettingViewModel : PaneViewModel
	{
		// Token: 0x06000270 RID: 624 RVA: 0x0000B938 File Offset: 0x00009B38
		public SettingViewModel()
		{
			base.Title = Application.Current.FindResource("menu_ViewSpecPropertyEditor") as string;
			base.ContentId = "SettingsPane";
			this.IsVisible = true;
			EditorWorkspace.This.ActiveDocumentChanged += this.This_ActiveDocumentChanged;
		}

		// Token: 0x06000271 RID: 625 RVA: 0x0000B98D File Offset: 0x00009B8D
		private void This_ActiveDocumentChanged(object sender, EventArgs e)
		{
			this.UI = ((EditorWorkspace.This.ActiveDocument == null) ? null : EditorWorkspace.This.ActiveDocument.PropertiesUI);
		}

		// Token: 0x17000098 RID: 152
		// (get) Token: 0x06000272 RID: 626 RVA: 0x0000B9B3 File Offset: 0x00009BB3
		// (set) Token: 0x06000273 RID: 627 RVA: 0x0000B9BB File Offset: 0x00009BBB
		public FrameworkElement UI
		{
			get
			{
				return this._ui;
			}
			set
			{
				this._ui = value;
				if (value != null)
				{
					base.ContentId = string.Format("{0}_{1}", base.Title, this._ui.GetType().Name);
				}
				base.RaisePropertyChanged("UI");
			}
		}

		// Token: 0x17000099 RID: 153
		// (get) Token: 0x06000274 RID: 628 RVA: 0x0000B9F8 File Offset: 0x00009BF8
		// (set) Token: 0x06000275 RID: 629 RVA: 0x0000BA00 File Offset: 0x00009C00
		public new bool IsVisible
		{
			get
			{
				return this._isVisible;
			}
			set
			{
				this._isVisible = value;
				base.RaisePropertyChanged("IsVisible");
			}
		}

		// Token: 0x04000169 RID: 361
		private FrameworkElement _ui;

		// Token: 0x0400016A RID: 362
		private bool _isVisible;
	}
}
