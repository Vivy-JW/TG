using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;

namespace SpecDesigner.SpecEditor.Controls
{
	// Token: 0x02000027 RID: 39
	public class ResetComboBox : ComboBox
	{
		// Token: 0x06000116 RID: 278 RVA: 0x00009916 File Offset: 0x00007B16
		static ResetComboBox()
		{
			FrameworkElement.DefaultStyleKeyProperty.OverrideMetadata(typeof(ResetComboBox), new FrameworkPropertyMetadata(typeof(ResetComboBox)));
		}

		// Token: 0x06000117 RID: 279 RVA: 0x0000993C File Offset: 0x00007B3C
		public override void OnApplyTemplate()
		{
			this.ResetImage = base.GetTemplateChild("resetImage") as Image;
			if (this.ResetImage != null)
			{
				this.ResetImage.MouseLeftButtonUp -= this.ResetImage_MouseLeftButtonUp;
				this.ResetImage.MouseLeftButtonUp += this.ResetImage_MouseLeftButtonUp;
			}
			base.OnApplyTemplate();
		}

		// Token: 0x06000118 RID: 280 RVA: 0x0000999B File Offset: 0x00007B9B
		private void ResetImage_MouseLeftButtonUp(object sender, MouseButtonEventArgs e)
		{
			base.SelectedValue = this.ResetText;
		}

		// Token: 0x17000021 RID: 33
		// (get) Token: 0x06000119 RID: 281 RVA: 0x000099A9 File Offset: 0x00007BA9
		// (set) Token: 0x0600011A RID: 282 RVA: 0x000099B1 File Offset: 0x00007BB1
		public string ResetText { get; set; }

		// Token: 0x04000093 RID: 147
		private Image ResetImage;
	}
}
