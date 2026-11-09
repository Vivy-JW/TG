using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Markup;
using SpecDesigner.CodeEditWindow.Helper;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000019 RID: 25
	public partial class GoToDiffLineWindow : Window
	{
		// Token: 0x1700001F RID: 31
		// (get) Token: 0x060000F9 RID: 249 RVA: 0x0000A50C File Offset: 0x0000870C
		// (set) Token: 0x060000FA RID: 250 RVA: 0x0000A51E File Offset: 0x0000871E
		public string RealLineOnCaret
		{
			get
			{
				return (string)base.GetValue(GoToDiffLineWindow.RealLineOnCaretProperty);
			}
			set
			{
				base.SetValue(GoToDiffLineWindow.RealLineOnCaretProperty, value);
			}
		}

		// Token: 0x060000FB RID: 251 RVA: 0x0000A52C File Offset: 0x0000872C
		public GoToDiffLineWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x060000FC RID: 252 RVA: 0x0000A53C File Offset: 0x0000873C
		public GoToDiffLineWindow(CodeTextEditor editor)
			: this()
		{
			this.Editor = editor;
			EditorDiffLineValidationRule editorDiffLineValidationRule = new EditorDiffLineValidationRule(this.Editor);
			Binding binding = new Binding("RealLineOnCaret")
			{
				UpdateSourceTrigger = UpdateSourceTrigger.PropertyChanged,
				ElementName = "This"
			};
			binding.ValidationRules.Add(editorDiffLineValidationRule);
			this.lineTextBox.SetBinding(TextBox.TextProperty, binding);
			this.RealLineOnCaret = editorDiffLineValidationRule.RealLineOnCaret.ToString();
			this.titleTextBlock.Text = string.Format(Application.Current.FindResource("CE_LineLimit") as string, this.Editor.Text.Replace("\a\r\n", "").Replace("\a\n", "").Split(new char[] { '\n' })
				.Length);
			base.Title = Application.Current.FindResource("CE_GoToLineTitle") as string;
			this.lineTextBox.Focus();
			this.lineTextBox.SelectAll();
			base.AddHandler(UIElement.PreviewKeyDownEvent, new KeyEventHandler(this.lineTextBox_KeyDown));
		}

		// Token: 0x060000FD RID: 253 RVA: 0x0000A664 File Offset: 0x00008864
		private void OKButton_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(true);
		}

		// Token: 0x060000FE RID: 254 RVA: 0x0000A672 File Offset: 0x00008872
		private void CancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(false);
		}

		// Token: 0x060000FF RID: 255 RVA: 0x0000A680 File Offset: 0x00008880
		private void lineTextBox_KeyDown(object sender, KeyEventArgs e)
		{
			Key key = e.Key;
			if (key == Key.Return)
			{
				e.Handled = true;
				this.OKButton_Click(null, new RoutedEventArgs());
				return;
			}
			if (key != Key.Escape)
			{
				return;
			}
			e.Handled = true;
			this.CancelButton_Click(null, new RoutedEventArgs());
		}

		// Token: 0x06000100 RID: 256 RVA: 0x0000A6C5 File Offset: 0x000088C5
		public void Dispose()
		{
			this.lineTextBox.ClearValue(TextBox.TextProperty);
			BindingOperations.ClearAllBindings(this);
		}

		// Token: 0x0400005D RID: 93
		public static readonly DependencyProperty RealLineOnCaretProperty = DependencyProperty.Register("RealLineOnCaret", typeof(string), typeof(GoToDiffLineWindow), new UIPropertyMetadata("1"));

		// Token: 0x0400005E RID: 94
		private CodeTextEditor Editor;
	}
}
