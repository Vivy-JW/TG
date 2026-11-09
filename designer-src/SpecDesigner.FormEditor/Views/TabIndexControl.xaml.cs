using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesigner.FormEditor.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x02000010 RID: 16
	public partial class TabIndexControl : UserControl
	{
		// Token: 0x06000051 RID: 81 RVA: 0x00003AA6 File Offset: 0x00001CA6
		public TabIndexControl()
		{
			this.InitializeComponent();
			base.Cursor = Cursors.Hand;
		}

		// Token: 0x06000052 RID: 82 RVA: 0x00003AC0 File Offset: 0x00001CC0
		protected override void OnMouseLeftButtonUp(MouseButtonEventArgs e)
		{
			e.Handled = true;
			XmlElement xmlElement = base.DataContext as XmlElement;
			if (xmlElement == null)
			{
				return;
			}
			if ((Keyboard.Modifiers & ModifierKeys.Control) == ModifierKeys.Control)
			{
				ComponentTabIndexService.Get(xmlElement.Key).SetAsCurrent(xmlElement);
				return;
			}
			ComponentTabIndexService.Get(xmlElement.Key).SetAsNext(xmlElement);
		}
	}
}
