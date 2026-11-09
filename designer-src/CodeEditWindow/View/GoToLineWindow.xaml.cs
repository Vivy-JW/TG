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
	// Token: 0x0200002A RID: 42
	public partial class GoToLineWindow : Window, IDisposable
	{
		// Token: 0x17000031 RID: 49
		// (get) Token: 0x06000183 RID: 387 RVA: 0x0000DDF4 File Offset: 0x0000BFF4
		// (set) Token: 0x06000184 RID: 388 RVA: 0x0000DE06 File Offset: 0x0000C006
		public string LineOnCaret
		{
			get
			{
				return (string)base.GetValue(GoToLineWindow.LineOnCaretProperty);
			}
			set
			{
				base.SetValue(GoToLineWindow.LineOnCaretProperty, value);
			}
		}

		// Token: 0x06000185 RID: 389 RVA: 0x0000DE14 File Offset: 0x0000C014
		private GoToLineWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x06000186 RID: 390 RVA: 0x0000DE24 File Offset: 0x0000C024
		public GoToLineWindow(CodeTextEditor editor)
			: this()
		{
			this.Editor = editor;
			EditorLineValidationRule editorLineValidationRule = new EditorLineValidationRule(this.Editor);
			Binding binding = new Binding("LineOnCaret")
			{
				UpdateSourceTrigger = UpdateSourceTrigger.PropertyChanged,
				ElementName = "This"
			};
			binding.ValidationRules.Add(editorLineValidationRule);
			this.lineTextBox.SetBinding(TextBox.TextProperty, binding);
			this.LineOnCaret = this.Editor.Document.GetLineByOffset(this.Editor.CaretOffset).LineNumber.ToString();
			this.titleTextBlock.Text = string.Format(Application.Current.FindResource("CE_LineLimit") as string, this.Editor.LineCount);
			base.Title = Application.Current.FindResource("CE_GoToLineTitle") as string;
			this.lineTextBox.Focus();
			this.lineTextBox.SelectAll();
			base.AddHandler(UIElement.PreviewKeyDownEvent, new KeyEventHandler(this.lineTextBox_KeyDown));
		}

		// Token: 0x06000187 RID: 391 RVA: 0x0000DF31 File Offset: 0x0000C131
		private void OKButton_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(true);
		}

		// Token: 0x06000188 RID: 392 RVA: 0x0000DF3F File Offset: 0x0000C13F
		private void CancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.DialogResult = new bool?(false);
		}

		// Token: 0x06000189 RID: 393 RVA: 0x0000DF50 File Offset: 0x0000C150
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

		// Token: 0x0600018A RID: 394 RVA: 0x0000DF95 File Offset: 0x0000C195
		public void Dispose()
		{
			this.lineTextBox.ClearValue(TextBox.TextProperty);
			BindingOperations.ClearAllBindings(this);
		}

		// Token: 0x040000AA RID: 170
		public static readonly DependencyProperty LineOnCaretProperty = DependencyProperty.Register("LineOnCaret", typeof(string), typeof(GoToLineWindow), new UIPropertyMetadata("1"));

		// Token: 0x040000AB RID: 171
		private CodeTextEditor Editor;
	}
}
