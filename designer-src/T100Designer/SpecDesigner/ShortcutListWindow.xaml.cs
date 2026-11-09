using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;

namespace SpecDesigner
{
	// Token: 0x02000018 RID: 24
	public partial class ShortcutListWindow : Window
	{
		// Token: 0x17000090 RID: 144
		// (get) Token: 0x060001EF RID: 495 RVA: 0x00008467 File Offset: 0x00006667
		public static ShortcutListWindow This
		{
			get
			{
				if (ShortcutListWindow._this == null)
				{
					ShortcutListWindow._this = new ShortcutListWindow();
				}
				return ShortcutListWindow._this;
			}
		}

		// Token: 0x060001F0 RID: 496 RVA: 0x0000847F File Offset: 0x0000667F
		private ShortcutListWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x060001F1 RID: 497 RVA: 0x0000848D File Offset: 0x0000668D
		public new void Show()
		{
			base.Show();
			if (base.Visibility != Visibility.Visible)
			{
				base.Visibility = Visibility.Visible;
			}
			base.Focus();
		}

		// Token: 0x060001F2 RID: 498 RVA: 0x000084AB File Offset: 0x000066AB
		protected override void OnClosing(CancelEventArgs e)
		{
			e.Cancel = true;
			base.Visibility = Visibility.Hidden;
		}

		// Token: 0x040000C0 RID: 192
		private static ShortcutListWindow _this;
	}
}
