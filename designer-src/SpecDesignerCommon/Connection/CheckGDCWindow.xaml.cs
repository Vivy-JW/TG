using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Drawing;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesigner.Controls.Controls;

namespace SpecDesignerCommon.Connection
{
	// Token: 0x020000E3 RID: 227
	public partial class CheckGDCWindow : Window, IDisposable
	{
		// Token: 0x17000222 RID: 546
		// (get) Token: 0x060007A5 RID: 1957 RVA: 0x000223F4 File Offset: 0x000205F4
		// (set) Token: 0x060007A6 RID: 1958 RVA: 0x000223FC File Offset: 0x000205FC
		public CheckOptionEnum CheckOption { get; private set; }

		// Token: 0x060007A7 RID: 1959 RVA: 0x00022408 File Offset: 0x00020608
		public CheckGDCWindow()
		{
			this.InitializeComponent();
			Icon warning = SystemIcons.Warning;
			this.Image_MessageBox.Source = warning.ToImageSource();
			this.cancelButton.Click += this.CancelButton_Click;
			this.tryAgainButton.Click += this.TryAgainButton_Click;
			this.continueButton.Click += this.ContinueButton_Click;
		}

		// Token: 0x060007A8 RID: 1960 RVA: 0x0002247D File Offset: 0x0002067D
		private void CancelButton_Click(object sender, RoutedEventArgs e)
		{
			this.CheckOption = CheckOptionEnum.Cancel;
			base.Close();
		}

		// Token: 0x060007A9 RID: 1961 RVA: 0x0002248C File Offset: 0x0002068C
		private void TryAgainButton_Click(object sender, RoutedEventArgs e)
		{
			this.CheckOption = CheckOptionEnum.TryAgain;
			base.Close();
		}

		// Token: 0x060007AA RID: 1962 RVA: 0x0002249B File Offset: 0x0002069B
		private void ContinueButton_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(true);
			this.CheckOption = CheckOptionEnum.Continue;
			base.Close();
		}

		// Token: 0x060007AB RID: 1963 RVA: 0x000224B8 File Offset: 0x000206B8
		public void Dispose()
		{
			this.cancelButton.Click -= this.CancelButton_Click;
			this.tryAgainButton.Click -= this.TryAgainButton_Click;
			this.continueButton.Click -= this.ContinueButton_Click;
		}
	}
}
