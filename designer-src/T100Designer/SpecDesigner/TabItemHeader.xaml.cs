using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using System.Windows.Media.Imaging;
using SpecDesigner.CodeEditWindow.View;
using SpecDesigner.FormEditor;

namespace SpecDesigner
{
	// Token: 0x0200000C RID: 12
	public partial class TabItemHeader : UserControl
	{
		// Token: 0x06000060 RID: 96 RVA: 0x0000318E File Offset: 0x0000138E
		private TabItemHeader()
		{
			this.InitializeComponent();
		}

		// Token: 0x06000061 RID: 97 RVA: 0x0000319C File Offset: 0x0000139C
		public TabItemHeader(UserControl userControl)
			: this()
		{
			if (userControl.GetType() == typeof(CodeEditorMainWindow))
			{
				this.title.Text = "CodeEditor";
				Uri uri = new Uri("/T100Designer;component/Images/document_4gl.png", UriKind.Relative);
				this.image.Source = new BitmapImage(uri);
				return;
			}
			if (userControl.GetType() == typeof(FormEditorMainWindow))
			{
				this.title.Text = "FormDesigner";
				Uri uri2 = new Uri("/T100Designer;component/Images/document_4fd.png", UriKind.Relative);
				this.image.Source = new BitmapImage(uri2);
				return;
			}
			if (userControl.GetType() == typeof(CodeSpecificationMainWindow))
			{
				this.title.Text = "CodeSpecEditor";
				Uri uri3 = new Uri("/T100Designer;component/Images/document_spec.png", UriKind.Relative);
				this.image.Source = new BitmapImage(uri3);
				return;
			}
			if (userControl.GetType() == typeof(ReportSpecificationWindow))
			{
				this.title.Text = "ReportSpecEditor";
				Uri uri4 = new Uri("/T100Designer;component/Images/document_spec.png", UriKind.Relative);
				this.image.Source = new BitmapImage(uri4);
			}
		}
	}
}
