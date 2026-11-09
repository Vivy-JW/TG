using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;

namespace SpecDesigner
{
	// Token: 0x02000029 RID: 41
	public partial class UploadSpecOptions : Window
	{
		// Token: 0x17000096 RID: 150
		// (get) Token: 0x06000257 RID: 599 RVA: 0x0000AF50 File Offset: 0x00009150
		// (set) Token: 0x06000258 RID: 600 RVA: 0x0000AF58 File Offset: 0x00009158
		public UploadSpecOptions.UploadTypes Selection { get; private set; }

		// Token: 0x06000259 RID: 601 RVA: 0x0000AF61 File Offset: 0x00009161
		private UploadSpecOptions()
		{
			this.InitializeComponent();
			base.Owner = Application.Current.MainWindow;
			base.WindowStartupLocation = WindowStartupLocation.CenterOwner;
			this.Selection = UploadSpecOptions.UploadTypes.SELF;
		}

		// Token: 0x0600025A RID: 602 RVA: 0x0000AF8D File Offset: 0x0000918D
		public UploadSpecOptions(string programName)
			: this()
		{
			this.textBlock_title.Text = string.Format(Application.Current.FindResource("Message_ConfirmBeforeUpload_SD") as string, programName);
		}

		// Token: 0x0600025B RID: 603 RVA: 0x0000AFBA File Offset: 0x000091BA
		private void UploadButton_Click(object sender, RoutedEventArgs e)
		{
			this.Selection = UploadSpecOptions.UploadTypes.SELF;
			base.DialogResult = new bool?(true);
			base.Close();
		}

		// Token: 0x0600025C RID: 604 RVA: 0x0000AFD5 File Offset: 0x000091D5
		private void UploadAndGenAllButton_Click(object sender, RoutedEventArgs e)
		{
			this.Selection = UploadSpecOptions.UploadTypes.ALL;
			base.DialogResult = new bool?(true);
			base.Close();
		}

		// Token: 0x0600025D RID: 605 RVA: 0x0000AFF0 File Offset: 0x000091F0
		private void CancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(false);
			base.Close();
		}

		// Token: 0x0200002A RID: 42
		public enum UploadTypes
		{
			// Token: 0x04000163 RID: 355
			[Description("")]
			ALL,
			// Token: 0x04000164 RID: 356
			[Description("tiptop")]
			SELF
		}
	}
}
