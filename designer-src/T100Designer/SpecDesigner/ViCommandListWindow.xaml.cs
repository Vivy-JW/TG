using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;

namespace SpecDesigner
{
	// Token: 0x02000030 RID: 48
	public partial class ViCommandListWindow : Window
	{
		// Token: 0x1700009A RID: 154
		// (get) Token: 0x0600027F RID: 639 RVA: 0x0000BB17 File Offset: 0x00009D17
		public static ViCommandListWindow This
		{
			get
			{
				if (ViCommandListWindow._this == null)
				{
					ViCommandListWindow._this = new ViCommandListWindow();
				}
				return ViCommandListWindow._this;
			}
		}

		// Token: 0x06000280 RID: 640 RVA: 0x0000BB2F File Offset: 0x00009D2F
		private ViCommandListWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x06000281 RID: 641 RVA: 0x0000BB3D File Offset: 0x00009D3D
		public new void Show()
		{
			base.Show();
			if (base.Visibility != Visibility.Visible)
			{
				base.Visibility = Visibility.Visible;
			}
			base.Focus();
		}

		// Token: 0x06000282 RID: 642 RVA: 0x0000BB5B File Offset: 0x00009D5B
		protected override void OnClosing(CancelEventArgs e)
		{
			e.Cancel = true;
			base.Visibility = Visibility.Hidden;
		}

		// Token: 0x0400016D RID: 365
		private static ViCommandListWindow _this;
	}
}
