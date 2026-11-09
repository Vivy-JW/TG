using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;

namespace SpecDesigner.SpecEditor.Controls
{
	// Token: 0x02000012 RID: 18
	public class ResetTextBox : TextBox
	{
		// Token: 0x06000072 RID: 114 RVA: 0x00006624 File Offset: 0x00004824
		static ResetTextBox()
		{
			FrameworkElement.DefaultStyleKeyProperty.OverrideMetadata(typeof(ResetTextBox), new FrameworkPropertyMetadata(typeof(ResetTextBox)));
		}

		// Token: 0x06000073 RID: 115 RVA: 0x00006684 File Offset: 0x00004884
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

		// Token: 0x06000074 RID: 116 RVA: 0x000066E4 File Offset: 0x000048E4
		private void ResetImage_MouseLeftButtonUp(object sender, MouseButtonEventArgs e)
		{
			base.Text = this.ResetText;
			BindingExpression bindingExpression = base.GetBindingExpression(TextBox.TextProperty);
			bindingExpression.UpdateSource();
		}

		// Token: 0x1700000B RID: 11
		// (get) Token: 0x06000075 RID: 117 RVA: 0x0000670F File Offset: 0x0000490F
		// (set) Token: 0x06000076 RID: 118 RVA: 0x00006721 File Offset: 0x00004921
		public string ResetText
		{
			get
			{
				return (string)base.GetValue(ResetTextBox.ResetTextProperty);
			}
			set
			{
				base.SetValue(ResetTextBox.ResetTextProperty, value);
			}
		}

		// Token: 0x0400004A RID: 74
		private Image ResetImage;

		// Token: 0x0400004B RID: 75
		public static readonly DependencyProperty ResetTextProperty = DependencyProperty.Register("ResetText", typeof(string), typeof(ResetTextBox), new PropertyMetadata(""));
	}
}
